// Package transfer implements chunked file transfer over the
// `transfer` data channel of every peer connection.
//
// Wire format (binary frames on the channel):
//
//	| type (1) | xfer_id (8 BE) | header_len (2 BE) | header_json | body... |
//
// Frame types:
//
//	0x10 START   — header carries Manifest{Name, Size, Mime}
//	0x11 CHUNK   — body carries one slice of the file (4096 bytes default)
//	0x12 END     — sender finished; receiver finalises
//	0x13 ABORT   — either side cancels; receiver discards partials
//	0x14 ACK     — receiver acknowledges N bytes (used by sender for
//	                progress backpressure, optional)
//
// Streams are identified by a 64-bit xfer_id chosen by the sender.
// Multiple concurrent transfers per peer are supported.
//
// Limitations (v1):
//   - Receiver buffers in memory before flushing to disk on END (we
//     don't implement file appending yet). Suitable for files up to
//     a few hundred MB; larger transfers should use the disk-streaming
//     variant added in a future revision.
//   - No resume on disconnect — a dropped channel restarts the file.
package transfer

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Frame types.
const (
	frameStart byte = 0x10
	frameChunk byte = 0x11
	frameEnd   byte = 0x12
	frameAbort byte = 0x13
	frameAck   byte = 0x14
)

// chunkSize is the per-frame payload size. WebRTC data channels limit
// individual messages to 64 KiB on most platforms; we stay well under
// to leave room for SCTP framing.
const chunkSize = 16 * 1024

// Manifest describes an offered file.
type Manifest struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Mime string `json:"mime,omitempty"`
}

// Mesh is the subset of mesh.Manager that the transfer engine needs.
type Mesh interface {
	SendTransferFrame(peerID string, payload []byte) error
}

// ProgressEvent describes an in-flight transfer's state for the UI.
type ProgressEvent struct {
	XferID    uint64    `json:"xferId"`
	PeerID    string    `json:"peerId"`
	Direction string    `json:"direction"` // "send" | "recv"
	Manifest  Manifest  `json:"manifest"`
	Bytes     int64     `json:"bytes"`
	Total     int64     `json:"total"`
	Done      bool      `json:"done"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"startedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	SavePath  string    `json:"savePath,omitempty"`
}

// ProgressFn is the callback the engine calls on every progress update.
// May be called from any goroutine; consumer is responsible for
// thread-safety on its side.
type ProgressFn func(ProgressEvent)

// Engine ties everything together. One per client.
type Engine struct {
	mesh    Mesh
	saveDir string
	logger  *slog.Logger
	onProg  ProgressFn

	mu      sync.Mutex
	nextID  uint64
	sends   map[key]*sendState
	recvs   map[key]*recvState
}

type key struct {
	peerID string
	xferID uint64
}

type sendState struct {
	manifest  Manifest
	startedAt time.Time
	bytesSent atomic.Int64
}

type recvState struct {
	manifest  Manifest
	startedAt time.Time
	buf       []byte
	bytesGot  int64
}

// NewEngine constructs a transfer engine. saveDir is where received
// files are written; if empty, ~/Downloads/Portal is used.
func NewEngine(mesh Mesh, saveDir string, logger *slog.Logger, onProg ProgressFn) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	if saveDir == "" {
		home, _ := os.UserHomeDir()
		saveDir = filepath.Join(home, "Downloads", "Portal")
	}
	_ = os.MkdirAll(saveDir, 0o755)
	return &Engine{
		mesh:    mesh,
		saveDir: saveDir,
		logger:  logger.With("component", "transfer"),
		onProg:  onProg,
		sends:   make(map[key]*sendState),
		recvs:   make(map[key]*recvState),
	}
}

// SaveDir returns the directory used for received files.
func (e *Engine) SaveDir() string { return e.saveDir }

// SendFile streams `path` to peerID. Returns the assigned xfer_id so
// the caller can match it against ProgressEvents. Errors here mean
// the start frame couldn't be sent; mid-flight failures are reported
// via ProgressEvent.Error.
func (e *Engine) SendFile(peerID, path string) (uint64, error) {
	st, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if st.IsDir() {
		return 0, errors.New("transfer: directories not supported")
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}

	xferID := e.nextXferID()
	manifest := Manifest{Name: filepath.Base(path), Size: st.Size(), Mime: ""}

	header, _ := json.Marshal(manifest)
	if err := e.sendFrame(peerID, frameStart, xferID, header, nil); err != nil {
		f.Close()
		return 0, err
	}
	state := &sendState{manifest: manifest, startedAt: time.Now()}
	e.mu.Lock()
	e.sends[key{peerID, xferID}] = state
	e.mu.Unlock()

	go e.pumpFile(peerID, xferID, f, state)
	return xferID, nil
}

// pumpFile streams the file's bytes, emitting progress callbacks as it
// goes. Closes f when done.
func (e *Engine) pumpFile(peerID string, xferID uint64, f *os.File, state *sendState) {
	defer f.Close()

	buf := make([]byte, chunkSize)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if errSend := e.sendFrame(peerID, frameChunk, xferID, nil, buf[:n]); errSend != nil {
				e.emit(ProgressEvent{
					XferID: xferID, PeerID: peerID, Direction: "send",
					Manifest: state.manifest, Bytes: state.bytesSent.Load(),
					Total: state.manifest.Size, Done: true,
					Error: errSend.Error(), StartedAt: state.startedAt, UpdatedAt: time.Now(),
				})
				return
			}
			state.bytesSent.Add(int64(n))
			e.emit(ProgressEvent{
				XferID: xferID, PeerID: peerID, Direction: "send",
				Manifest: state.manifest, Bytes: state.bytesSent.Load(),
				Total: state.manifest.Size, Done: false,
				StartedAt: state.startedAt, UpdatedAt: time.Now(),
			})
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = e.sendFrame(peerID, frameAbort, xferID, nil, []byte(err.Error()))
			e.emit(ProgressEvent{
				XferID: xferID, PeerID: peerID, Direction: "send",
				Manifest: state.manifest, Bytes: state.bytesSent.Load(),
				Total: state.manifest.Size, Done: true,
				Error: err.Error(), StartedAt: state.startedAt, UpdatedAt: time.Now(),
			})
			return
		}
	}
	_ = e.sendFrame(peerID, frameEnd, xferID, nil, nil)
	e.mu.Lock()
	delete(e.sends, key{peerID, xferID})
	e.mu.Unlock()
	e.emit(ProgressEvent{
		XferID: xferID, PeerID: peerID, Direction: "send",
		Manifest: state.manifest, Bytes: state.bytesSent.Load(),
		Total: state.manifest.Size, Done: true,
		StartedAt: state.startedAt, UpdatedAt: time.Now(),
	})
}

// HandleFrame is called by mesh.Manager for every inbound frame on the
// transfer channel from peerID.
func (e *Engine) HandleFrame(peerID string, payload []byte) {
	if len(payload) < 11 {
		return
	}
	t := payload[0]
	xferID := binary.BigEndian.Uint64(payload[1:9])
	hdrLen := binary.BigEndian.Uint16(payload[9:11])
	if int(11+hdrLen) > len(payload) {
		return
	}
	header := payload[11 : 11+hdrLen]
	body := payload[11+hdrLen:]

	switch t {
	case frameStart:
		var m Manifest
		if err := json.Unmarshal(header, &m); err != nil {
			return
		}
		e.mu.Lock()
		e.recvs[key{peerID, xferID}] = &recvState{manifest: m, startedAt: time.Now()}
		e.mu.Unlock()
		e.emit(ProgressEvent{
			XferID: xferID, PeerID: peerID, Direction: "recv",
			Manifest: m, Bytes: 0, Total: m.Size, Done: false,
			StartedAt: time.Now(), UpdatedAt: time.Now(),
		})

	case frameChunk:
		e.mu.Lock()
		st := e.recvs[key{peerID, xferID}]
		if st == nil {
			e.mu.Unlock()
			return
		}
		st.buf = append(st.buf, body...)
		st.bytesGot += int64(len(body))
		manifest := st.manifest
		bytesGot := st.bytesGot
		startedAt := st.startedAt
		e.mu.Unlock()
		e.emit(ProgressEvent{
			XferID: xferID, PeerID: peerID, Direction: "recv",
			Manifest: manifest, Bytes: bytesGot, Total: manifest.Size, Done: false,
			StartedAt: startedAt, UpdatedAt: time.Now(),
		})

	case frameEnd:
		e.mu.Lock()
		st := e.recvs[key{peerID, xferID}]
		delete(e.recvs, key{peerID, xferID})
		e.mu.Unlock()
		if st == nil {
			return
		}
		// Save to disk, choosing a non-clobbering path.
		safe := safeName(st.manifest.Name)
		path := filepath.Join(e.saveDir, safe)
		path = ensureUniquePath(path)
		err := os.WriteFile(path, st.buf, 0o644)
		ev := ProgressEvent{
			XferID: xferID, PeerID: peerID, Direction: "recv",
			Manifest: st.manifest, Bytes: st.bytesGot, Total: st.manifest.Size, Done: true,
			StartedAt: st.startedAt, UpdatedAt: time.Now(), SavePath: path,
		}
		if err != nil {
			ev.Error = err.Error()
			ev.SavePath = ""
		}
		e.emit(ev)

	case frameAbort:
		e.mu.Lock()
		st := e.recvs[key{peerID, xferID}]
		delete(e.recvs, key{peerID, xferID})
		e.mu.Unlock()
		if st != nil {
			e.emit(ProgressEvent{
				XferID: xferID, PeerID: peerID, Direction: "recv",
				Manifest: st.manifest, Bytes: st.bytesGot, Total: st.manifest.Size,
				Done: true, Error: string(body), StartedAt: st.startedAt, UpdatedAt: time.Now(),
			})
		}
	}
}

// CancelSend tells the receiver to discard a transfer in-flight.
func (e *Engine) CancelSend(peerID string, xferID uint64) {
	_ = e.sendFrame(peerID, frameAbort, xferID, nil, []byte("cancelled by sender"))
	e.mu.Lock()
	delete(e.sends, key{peerID, xferID})
	e.mu.Unlock()
}

// ----------------------------------------------------------------------------

func (e *Engine) nextXferID() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nextID++
	if e.nextID == 0 {
		e.nextID = 1
	}
	return e.nextID
}

func (e *Engine) sendFrame(peerID string, t byte, xferID uint64, header, body []byte) error {
	hdrLen := len(header)
	if hdrLen > 0xFFFF {
		return fmt.Errorf("transfer: header too large")
	}
	buf := make([]byte, 11+hdrLen+len(body))
	buf[0] = t
	binary.BigEndian.PutUint64(buf[1:9], xferID)
	binary.BigEndian.PutUint16(buf[9:11], uint16(hdrLen))
	copy(buf[11:], header)
	copy(buf[11+hdrLen:], body)
	return e.mesh.SendTransferFrame(peerID, buf)
}

func (e *Engine) emit(ev ProgressEvent) {
	if e.onProg != nil {
		e.onProg(ev)
	}
}

func safeName(name string) string {
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c == '/' || c == '\\' || c == ':' || c == '\x00':
			out = append(out, '_')
		default:
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return "untitled"
	}
	return string(out)
}

func ensureUniquePath(p string) string {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	dir := filepath.Dir(p)
	base := filepath.Base(p)
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]
	for i := 1; i < 1000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return p
}
