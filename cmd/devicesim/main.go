// Command devicesim is a laptop stand-in for the ESP32: it speaks the exact
// binary MQTT wire protocol (the reference the firmware mirrors), sending a WAV
// file as sequenced chunks and reassembling the binary response.
//
// Usage:
//
//	go run ./cmd/devicesim -wav testdata/audio/sample.wav -broker tcp://localhost:1883
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"log"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	device := flag.String("device", "test-device", "device id (goes in the topic)")
	kid := flag.String("kid", "KID001", "kid id")
	wavPath := flag.String("wav", "", "input WAV (16kHz/16-bit/mono) [required]")
	chunkBytes := flag.Int("chunk", 32000, "chunk size in bytes (~1s of audio)")
	out := flag.String("out", "combined.pcm", "output PCM file for the response")
	timeout := flag.Duration("timeout", 30*time.Second, "response wait timeout")
	flag.Parse()

	if *wavPath == "" {
		log.Fatal("missing -wav")
	}

	pcm, err := os.ReadFile(*wavPath)
	if err != nil {
		log.Fatal(err)
	}
	if len(pcm) > 44 {
		pcm = pcm[44:] // strip the 44-byte WAV header → raw PCM
	}

	var (
		mu         sync.Mutex
		respChunks = map[int][]byte{}
		respTotal  = -1
		once       sync.Once
		done       = make(chan struct{})
	)
	finish := func() { once.Do(func() { close(done) }) }

	opts := mqtt.NewClientOptions().AddBroker(*broker).SetClientID("cwlb-devicesim-" + *device)
	client := mqtt.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		log.Fatal("connect: ", tok.Error())
	}
	defer client.Disconnect(250)

	base := "/device/" + *device + "/audio/"

	// subscribe BEFORE publishing so we don't miss the response
	client.Subscribe(base+"response_chunk", 1, func(_ mqtt.Client, m mqtt.Message) {
		p := m.Payload()
		if len(p) < 4 {
			return
		}
		idx := int(binary.LittleEndian.Uint16(p[0:2]))
		tot := int(binary.LittleEndian.Uint16(p[2:4]))
		data := make([]byte, len(p)-4)
		copy(data, p[4:])

		mu.Lock()
		respChunks[idx] = data
		respTotal = tot
		complete := len(respChunks) == respTotal
		mu.Unlock()

		log.Printf("<- response chunk %d/%d (%d bytes)", idx+1, tot, len(data))
		if complete {
			finish()
		}
	})
	client.Subscribe(base+"response_end", 1, func(_ mqtt.Client, _ mqtt.Message) {
		log.Println("<- response_end")
		mu.Lock()
		complete := respTotal >= 0 && len(respChunks) == respTotal
		mu.Unlock()
		if complete {
			finish()
		}
	})

	// start
	startPayload, _ := json.Marshal(map[string]string{"kid_id": *kid})
	publish(client, base+"start", startPayload)
	log.Printf("-> start (device=%s kid=%s)", *device, *kid)

	// chunks
	total := (len(pcm) + *chunkBytes - 1) / *chunkBytes
	for i := range total {
		s := i * *chunkBytes
		e := s + *chunkBytes
		if e > len(pcm) {
			e = len(pcm)
		}
		publish(client, base+"chunk", encodeChunk(i, total, pcm[s:e]))
		log.Printf("-> chunk %d/%d (%d bytes)", i+1, total, e-s)
	}

	// end
	publish(client, base+"end", []byte{})
	log.Printf("-> end. waiting up to %s for response...", *timeout)

	select {
	case <-done:
	case <-time.After(*timeout):
		log.Println("timeout waiting for response")
	}

	mu.Lock()
	defer mu.Unlock()
	if respTotal <= 0 {
		log.Fatal("no response received")
	}

	var combined []byte
	missing := 0
	for i := range respTotal {
		c, ok := respChunks[i]
		if !ok {
			missing++
			continue
		}
		combined = append(combined, c...)
	}
	if missing > 0 {
		log.Printf("WARNING: %d/%d response chunks missing", missing, respTotal)
	}
	if err := os.WriteFile(*out, combined, 0644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s (%d bytes, %d/%d chunks) — play: aplay -r 16000 -f S16_LE %s",
		*out, len(combined), respTotal-missing, respTotal, *out)
}

func publish(c mqtt.Client, topic string, payload []byte) {
	if tok := c.Publish(topic, 1, false, payload); tok.Wait() && tok.Error() != nil {
		log.Printf("publish %s: %v", topic, tok.Error())
	}
}

// encodeChunk frames a chunk as [u16 index LE][u16 total LE][PCM] — the same wire
// format the server parses and the firmware must emit.
func encodeChunk(index, total int, pcm []byte) []byte {
	buf := make([]byte, 4+len(pcm))
	binary.LittleEndian.PutUint16(buf[0:2], uint16(index))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(total))
	copy(buf[4:], pcm)
	return buf
}
