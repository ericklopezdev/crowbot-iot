package mqtt

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

func TestSessionRouter_SingleSession(t *testing.T) {
	r := newSessionRouter()
	r.start("dev-1", "kid-1")

	data := []byte("audio-data")
	if err := r.addChunk("dev-1", 0, 1, data); err != nil {
		t.Fatal(err)
	}

	full, err := r.complete("dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(full, data) {
		t.Errorf("got %q want %q", full, data)
	}
}

func TestSessionRouter_ReassemblesInOrder(t *testing.T) {
	r := newSessionRouter()
	r.start("dev-1", "kid-1")

	chunks := [][]byte{[]byte("first"), []byte("second"), []byte("third")}
	// deliver out of order: 2, 0, 1
	for _, i := range []int{2, 0, 1} {
		if err := r.addChunk("dev-1", i, 3, chunks[i]); err != nil {
			t.Fatal(err)
		}
	}

	full, err := r.complete("dev-1")
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Join(chunks, nil)
	if !bytes.Equal(full, want) {
		t.Errorf("got %q want %q", full, want)
	}
}

func TestSessionRouter_DetectsMissingChunk(t *testing.T) {
	r := newSessionRouter()
	r.start("dev-1", "kid-1")

	// send only chunk 0 out of 2
	if err := r.addChunk("dev-1", 0, 2, []byte("data")); err != nil {
		t.Fatal(err)
	}

	if _, err := r.complete("dev-1"); err == nil {
		t.Error("expected error for missing chunk 1")
	}
}

func TestSessionRouter_ConcurrentSessions(t *testing.T) {
	r := newSessionRouter()
	const n = 20
	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			deviceID := fmt.Sprintf("device-%d", i)
			payload := []byte(fmt.Sprintf("audio-%d", i))

			r.start(deviceID, "kid-1")
			if err := r.addChunk(deviceID, 0, 1, payload); err != nil {
				t.Errorf("device %s addChunk: %v", deviceID, err)
				return
			}
			full, err := r.complete(deviceID)
			if err != nil {
				t.Errorf("device %s complete: %v", deviceID, err)
				return
			}
			if !bytes.Equal(full, payload) {
				t.Errorf("device %s: got %q want %q", deviceID, full, payload)
			}
		}(i)
	}
	wg.Wait()
}

func TestSessionRouter_NoSessionOnChunk(t *testing.T) {
	r := newSessionRouter()
	if err := r.addChunk("ghost", 0, 1, []byte("data")); err == nil {
		t.Error("expected error for chunk without session")
	}
}

func TestChunkFraming_RoundTrip(t *testing.T) {
	pcm := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	encoded := encodeChunk(7, 42, pcm)

	index, total, got, err := parseChunk(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if index != 7 || total != 42 {
		t.Errorf("got index=%d total=%d want 7/42", index, total)
	}
	if !bytes.Equal(got, pcm) {
		t.Errorf("got pcm %v want %v", got, pcm)
	}
}

func TestParseChunk_TooShort(t *testing.T) {
	if _, _, _, err := parseChunk([]byte{0x00, 0x01}); err == nil {
		t.Error("expected error for payload shorter than 4 bytes")
	}
}

func TestDeviceIDFromTopic(t *testing.T) {
	cases := map[string]struct {
		want string
		ok   bool
	}{
		"/device/crowbot-01/audio/chunk": {"crowbot-01", true},
		"/device/dev-2/audio/start":      {"dev-2", true},
		"/device/audio/chunk":            {"", false},
		"/other/x/audio/chunk":           {"", false},
		"garbage":                        {"", false},
	}
	for topic, exp := range cases {
		got, ok := deviceIDFromTopic(topic)
		if got != exp.want || ok != exp.ok {
			t.Errorf("%s: got (%q,%v) want (%q,%v)", topic, got, ok, exp.want, exp.ok)
		}
	}
}
