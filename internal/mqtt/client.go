package mqtt

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/ErickLopezDev/cwlb-server/internal/core"
	"github.com/ErickLopezDev/cwlb-server/internal/store"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Wire format (device id travels in the topic, not the payload):
//   /device/{id}/audio/start  JSON  {"kid_id":"..."}
//   /device/{id}/audio/chunk  binary [u16 index LE][u16 total LE][PCM 16-bit]
//   /device/{id}/audio/end    empty
//
// Response (server -> device), same binary chunk framing:
//   /device/{id}/audio/response_chunk  binary [u16 index][u16 total][PCM]
//   /device/{id}/audio/response_end    empty

type startPayload struct {
	KidID string `json:"kid_id"`
}

// -- session --

type audioSession struct {
	mu     sync.Mutex
	chunks map[int][]byte
	total  int
	kidID  string
}

// -- session router (per-device, concurrency-safe) --

type sessionRouter struct {
	mu       sync.Mutex
	sessions map[string]*audioSession
}

func newSessionRouter() *sessionRouter {
	return &sessionRouter{sessions: make(map[string]*audioSession)}
}

func (r *sessionRouter) start(deviceID, kidID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[deviceID] = &audioSession{chunks: make(map[int][]byte), kidID: kidID}
	log.Printf("[mqtt] session started: device=%s kid=%s", deviceID, kidID)
}

func (r *sessionRouter) addChunk(deviceID string, index, total int, data []byte) error {
	r.mu.Lock()
	sess, ok := r.sessions[deviceID]
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("no active session for device %s", deviceID)
	}

	// copy: the MQTT library may reuse the payload buffer after the callback returns
	cp := make([]byte, len(data))
	copy(cp, data)

	sess.mu.Lock()
	defer sess.mu.Unlock()
	sess.chunks[index] = cp
	sess.total = total
	log.Printf("[mqtt] chunk %d/%d device=%s (%d bytes)", index+1, total, deviceID, len(data))
	return nil
}

// complete removes the session, verifies all chunks are present, and returns the
// reassembled audio in index order. Errors if any chunk is missing.
func (r *sessionRouter) complete(deviceID string) ([]byte, error) {
	r.mu.Lock()
	sess, ok := r.sessions[deviceID]
	delete(r.sessions, deviceID)
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no active session for device %s", deviceID)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	var missing []int
	for i := range sess.total {
		if _, ok := sess.chunks[i]; !ok {
			missing = append(missing, i)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("device %s: missing chunk indices %v (got %d/%d)", deviceID, missing, len(sess.chunks), sess.total)
	}

	full := make([]byte, 0, sess.total*2048)
	for i := range sess.total {
		full = append(full, sess.chunks[i]...)
	}
	log.Printf("[mqtt] session complete: device=%s %d bytes", deviceID, len(full))
	return full, nil
}

// -- binary chunk framing --

func parseChunk(payload []byte) (index, total int, pcm []byte, err error) {
	if len(payload) < 4 {
		return 0, 0, nil, fmt.Errorf("chunk too short: %d bytes", len(payload))
	}
	index = int(binary.LittleEndian.Uint16(payload[0:2]))
	total = int(binary.LittleEndian.Uint16(payload[2:4]))
	pcm = payload[4:]
	return index, total, pcm, nil
}

func encodeChunk(index, total int, pcm []byte) []byte {
	buf := make([]byte, 4+len(pcm))
	binary.LittleEndian.PutUint16(buf[0:2], uint16(index))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(total))
	copy(buf[4:], pcm)
	return buf
}

// deviceIDFromTopic extracts {id} from /device/{id}/audio/{action}.
func deviceIDFromTopic(topic string) (string, bool) {
	parts := strings.Split(topic, "/")
	if len(parts) >= 5 && parts[1] == "device" && parts[3] == "audio" {
		return parts[2], true
	}
	return "", false
}

// -- MQTT client --

// NewClient connects to the broker and wires the audio handlers. queries may be
// nil, in which case interactions are processed but not persisted.
func NewClient(ctx context.Context, broker string, orchestrator *core.Orchestrator, queries store.Querier) mqtt.Client {
	router := newSessionRouter()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID("cwlb-mqtt-server")
	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		log.Printf("[mqtt] connection lost: %v", err)
	}

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("[mqtt] connected to broker")

		sub := func(topic string, h mqtt.MessageHandler) {
			if tok := c.Subscribe(topic, 1, h); tok.Wait() && tok.Error() != nil {
				log.Printf("[mqtt] subscribe %s: %v", topic, tok.Error())
			}
		}

		sub("/device/+/audio/start", func(_ mqtt.Client, msg mqtt.Message) {
			handleStart(router, msg)
		})
		sub("/device/+/audio/chunk", func(_ mqtt.Client, msg mqtt.Message) {
			handleChunk(router, msg)
		})
		sub("/device/+/audio/end", func(c mqtt.Client, msg mqtt.Message) {
			handleEnd(ctx, c, router, orchestrator, queries, msg)
		})

		log.Println("[mqtt] subscribed to /device/+/audio/{start,chunk,end}")
	}

	client := mqtt.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		log.Fatal("[mqtt] connect failed: ", tok.Error())
	}
	return client
}

func handleStart(router *sessionRouter, msg mqtt.Message) {
	deviceID, ok := deviceIDFromTopic(msg.Topic())
	if !ok {
		log.Printf("[mqtt] bad start topic: %s", msg.Topic())
		return
	}
	var p startPayload
	if len(msg.Payload()) > 0 {
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("[mqtt] bad start payload for %s: %v", deviceID, err)
		}
	}
	router.start(deviceID, p.KidID)
}

func handleChunk(router *sessionRouter, msg mqtt.Message) {
	deviceID, ok := deviceIDFromTopic(msg.Topic())
	if !ok {
		log.Printf("[mqtt] bad chunk topic: %s", msg.Topic())
		return
	}
	index, total, pcm, err := parseChunk(msg.Payload())
	if err != nil {
		log.Printf("[mqtt] bad chunk for %s: %v", deviceID, err)
		return
	}
	if err := router.addChunk(deviceID, index, total, pcm); err != nil {
		log.Printf("[mqtt] addChunk: %v", err)
	}
}

func handleEnd(ctx context.Context, client mqtt.Client, router *sessionRouter, orchestrator *core.Orchestrator, queries store.Querier, msg mqtt.Message) {
	deviceID, ok := deviceIDFromTopic(msg.Topic())
	if !ok {
		log.Printf("[mqtt] bad end topic: %s", msg.Topic())
		return
	}
	go processSession(ctx, client, router, orchestrator, queries, deviceID)
}

func processSession(ctx context.Context, client mqtt.Client, router *sessionRouter, orchestrator *core.Orchestrator, queries store.Querier, deviceID string) {
	fullAudio, err := router.complete(deviceID)
	if err != nil {
		log.Printf("[mqtt] session %s: %v", deviceID, err)
		return
	}

	result, err := orchestrator.HandleAudio(ctx, fullAudio)
	if err != nil {
		log.Printf("[mqtt] pipeline error for %s: %v", deviceID, err)
		return
	}

	publishResponse(client, deviceID, result.Audio)
	persistInteraction(ctx, queries, deviceID, result)
}

// persistInteraction stores one turn. It is best-effort: any failure is logged
// and never blocks the voice response already sent to the child.
func persistInteraction(ctx context.Context, queries store.Querier, deviceID string, r *core.TurnResult) {
	if queries == nil {
		return
	}

	devUUID, err := uuid.Parse(deviceID)
	if err != nil {
		log.Printf("[mqtt] skip persist: device id %q is not a uuid: %v", deviceID, err)
		return
	}

	// Resolve device -> active child for attribution. No assignment is fine:
	// the interaction is stored with a null child_id.
	var childID pgtype.UUID
	if child, err := queries.GetActiveChildForDevice(ctx, devUUID); err != nil {
		log.Printf("[mqtt] no active child for device %s: %v", deviceID, err)
	} else {
		childID = pgtype.UUID{Bytes: child.ID, Valid: true}
	}

	if _, err := queries.CreateInteraction(ctx, store.CreateInteractionParams{
		DeviceID:       devUUID,
		ChildID:        childID,
		SessionID:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Transcript:     r.Transcript,
		ResponseText:   r.ResponseText,
		SttLatencyMs:   r.STTLatencyMs,
		LlmLatencyMs:   r.LLMLatencyMs,
		TtsLatencyMs:   r.TTSLatencyMs,
		TotalLatencyMs: r.TotalLatencyMs,
	}); err != nil {
		log.Printf("[mqtt] persist interaction for %s failed: %v", deviceID, err)
		return
	}
	log.Printf("[mqtt] interaction persisted: device=%s child_valid=%t", deviceID, childID.Valid)
}

func publishResponse(client mqtt.Client, deviceID string, audio []byte) {
	const chunkSize = 32000
	total := (len(audio) + chunkSize - 1) / chunkSize

	for i := range total {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(audio) {
			end = len(audio)
		}
		payload := encodeChunk(i, total, audio[start:end])
		client.Publish("/device/"+deviceID+"/audio/response_chunk", 1, false, payload)
	}
	client.Publish("/device/"+deviceID+"/audio/response_end", 1, false, []byte{})
	log.Printf("[mqtt] response published: device=%s %d chunks", deviceID, total)
}
