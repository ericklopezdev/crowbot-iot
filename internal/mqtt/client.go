package mqtt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/ErickLopezDev/cwlb-server/internal/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type AudioSession struct {
	mu     sync.Mutex
	chunks [][]byte
}

type StartMessage struct {
	RobotID string `json:"robot_id"`
	KidID   string `json:"kid_id"`
}

func NewClient(broker string, orchestrator *core.Orchestrator) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID("cwlb-mqtt-server")

	sessions := make(map[string]*AudioSession)
	var sessionsMu sync.Mutex
	var currentDevice string

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("Connected to MQTT broker")

		if token := c.Subscribe("/device/audio/start", 0,
			func(_ mqtt.Client, msg mqtt.Message) {
				handleStart(&sessionsMu, sessions, &currentDevice, msg)
			},
		); token.Wait() && token.Error() != nil {
			log.Println("Subscribe start failed:", token.Error())
		}

		if token := c.Subscribe("/device/audio/chunk", 0,
			func(_ mqtt.Client, msg mqtt.Message) {
				handleChunk(&sessionsMu, sessions, &currentDevice, msg)
			},
		); token.Wait() && token.Error() != nil {
			log.Println("Subscribe chunk failed:", token.Error())
		}

		if token := c.Subscribe("/device/audio/end", 0,
			func(client mqtt.Client, msg mqtt.Message) {
				handleEnd(&sessionsMu, sessions, &currentDevice, client, orchestrator, msg)
			},
		); token.Wait() && token.Error() != nil {
			log.Println("Subscribe end failed:", token.Error())
		}

		log.Println("Subscribed to MQTT topics")
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	return client
}

// Starts a new audio session
func handleStart(mu *sync.Mutex, sessions map[string]*AudioSession, currentDevice *string, msg mqtt.Message) {
	var startMsg StartMessage
	if err := json.Unmarshal(msg.Payload(), &startMsg); err != nil {
		log.Println("Error parsing start message:", err)
		return
	}
	deviceID := startMsg.RobotID
	*currentDevice = deviceID
	mu.Lock()
	sessions[deviceID] = &AudioSession{}
	mu.Unlock()
	log.Println("Recording started for", deviceID)
}

// Saves a chunk of audio
func handleChunk(mu *sync.Mutex, sessions map[string]*AudioSession, currentDevice *string, msg mqtt.Message) {
	data, err := base64.StdEncoding.DecodeString(string(msg.Payload()))
	if err != nil {
		log.Println("Error decoding chunk:", err)
		return
	}

	deviceID := *currentDevice
	if deviceID == "" {
		log.Println("No current device set")
		return
	}

	mu.Lock()
	session, ok := sessions[deviceID]
	mu.Unlock()
	if !ok {
		log.Println("No active session for", deviceID)
		return
	}

	session.mu.Lock()
	session.chunks = append(session.chunks, data)
	session.mu.Unlock()

	log.Println("Chunk received for", deviceID, "size:", len(data))
}

// Combines and process the audio, then publish the response in chunks
func handleEnd(mu *sync.Mutex, sessions map[string]*AudioSession, currentDevice *string, client mqtt.Client, orchestrator *core.Orchestrator, msg mqtt.Message) {
	deviceID := *currentDevice
	if deviceID == "" {
		log.Println("No current device set")
		return
	}

	mu.Lock()
	session := sessions[deviceID]
	delete(sessions, deviceID)
	mu.Unlock()

	if session == nil {
		log.Println("Error: no active session for", deviceID)
		return
	}

	fullAudio := []byte{}
	for _, chunk := range session.chunks {
		fullAudio = append(fullAudio, chunk...)
	}

	audioResponse, err := orchestrator.HandleAudio(fullAudio)
	if err != nil {
		log.Println("Error processing audio:", err)
		return
	}

	// Send response in chunks
	chunkSize := 32000 // ~1 second at 16000 Hz, 16-bit
	totalChunks := (len(audioResponse) + chunkSize - 1) / chunkSize
	for i := range totalChunks {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(audioResponse) {
			end = len(audioResponse)
		}
		chunk := audioResponse[start:end]
		encoded := base64.StdEncoding.EncodeToString(chunk)
		payload := fmt.Sprintf(`{"index": %d, "total": %d, "data": "%s"}`, i, totalChunks, encoded)
		client.Publish("/device/"+deviceID+"/audio/response_chunk", 0, false, payload)
	}
	client.Publish("/device/"+deviceID+"/audio/response_end", 0, false, "")
	log.Println("Response chunks published to", deviceID)
}

func extractDeviceID(topic string) string {
	// example: /device/crowbot-01/audio/chunk
	parts := strings.Split(topic, "/")
	if len(parts) >= 3 {
		return parts[2] // "crowbot-01"
	}
	return "unknown"
}
