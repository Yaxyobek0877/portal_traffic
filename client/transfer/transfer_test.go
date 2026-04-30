package transfer

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Loopback mesh: in-process delivery of frames between two engines.
type fake struct {
	peerID string
	other  *Engine
	queue  chan []byte
	done   chan struct{}
}

func newFake(id string) *fake {
	return &fake{
		peerID: id,
		queue:  make(chan []byte, 1024),
		done:   make(chan struct{}),
	}
}

func (f *fake) start() {
	go func() {
		for {
			select {
			case <-f.done:
				return
			case buf := <-f.queue:
				f.other.HandleFrame(f.peerID, buf)
			}
		}
	}()
}

func (f *fake) stop() {
	select {
	case <-f.done:
	default:
		close(f.done)
	}
}

func (f *fake) SendTransferFrame(_ string, payload []byte) error {
	cp := make([]byte, len(payload))
	copy(cp, payload)
	select {
	case f.queue <- cp:
	case <-f.done:
	}
	return nil
}

func TestTransferRoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "alice.bin")
	contents := make([]byte, 256*1024) // ≈16 chunks
	if _, err := rand.Read(contents); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, contents, 0o644); err != nil {
		t.Fatal(err)
	}

	saveDir := filepath.Join(dir, "saved")

	var (
		mu       sync.Mutex
		latest   ProgressEvent
		recvDone atomic.Bool
		sendDone atomic.Bool
	)
	progress := func(ev ProgressEvent) {
		mu.Lock()
		latest = ev
		mu.Unlock()
		if ev.Done && ev.Error == "" {
			if ev.Direction == "recv" {
				recvDone.Store(true)
			} else {
				sendDone.Store(true)
			}
		}
	}

	// Sender side has no save dir; receiver writes to saveDir.
	fakeA := newFake("a")
	fakeB := newFake("b")
	a := NewEngine(fakeA, "", nil, progress)
	b := NewEngine(fakeB, saveDir, nil, progress)
	fakeA.other = b
	fakeB.other = a
	fakeA.start()
	fakeB.start()
	t.Cleanup(func() { fakeA.stop(); fakeB.stop() })

	xferID, err := a.SendFile("b", src)
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !(recvDone.Load() && sendDone.Load()) {
		time.Sleep(20 * time.Millisecond)
	}
	if !recvDone.Load() || !sendDone.Load() {
		t.Fatalf("transfer did not complete (recv=%v send=%v latest=%+v)",
			recvDone.Load(), sendDone.Load(), latest)
	}

	mu.Lock()
	got := latest
	mu.Unlock()
	if got.SavePath == "" {
		t.Fatalf("no save path in event: %+v", got)
	}
	saved, err := os.ReadFile(got.SavePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(saved, contents) {
		t.Errorf("saved bytes (%d) ≠ source (%d)", len(saved), len(contents))
	}
	if xferID == 0 {
		t.Error("xfer id should be non-zero")
	}
}
