package mobile

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// ConnectionCapture owns one transport generation, independent of drive sessions.
// Native callbacks must be serialized; chunks have no protocol framing meaning.
type ConnectionCapture struct {
	mu                                    sync.Mutex
	file                                  *os.File
	path, id                              string
	anchorNS, anchorMS, lastNS, seq, size int64
	received, accepted, requested         int64
	pending                               []byte
	used, closed                          bool
	failure                               error
	recent                                []captureRow
}
type captureRow struct {
	Version  int             `json:"capture_version"`
	ID       string          `json:"connection_id"`
	Sequence int64           `json:"sequence"`
	Kind     string          `json:"kind"`
	NS       int64           `json:"native_monotonic_ns"`
	MS       int64           `json:"mapped_unix_ms"`
	Bytes    []byte          `json:"bytes_base64,omitempty"`
	Detail   string          `json:"detail,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// NewConnectionCapture uses native continuous monotonic nanoseconds and a Unix
// millisecond anchor. Files are transport evidence, never fabricated drive logs.
func NewConnectionCapture(root, metadata string, unixMS, nativeNS int64) (_ *ConnectionCapture, err error) {
	defer wrapFailure("NewConnectionCapture", &err)
	var obj map[string]any
	if !filepath.IsAbs(root) || unixMS <= 0 || nativeNS < 0 || len(metadata) > 8192 || json.Unmarshal([]byte(metadata), &obj) != nil || obj == nil {
		return nil, fault("invalid_input", "NewConnectionCapture", fmt.Errorf("absolute root, valid clocks and bounded object metadata required"))
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	id := "connection-" + hex.EncodeToString(nonce[:])
	path := filepath.Join(root, id+".jsonl")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	c := &ConnectionCapture{file: f, path: path, id: id, anchorNS: nativeNS, anchorMS: unixMS, lastNS: nativeNS}
	if err = c.append("opened", nil, "transport only", json.RawMessage(metadata), nativeNS); err != nil {
		_ = f.Close()
		return nil, err
	}
	return c, nil
}
func (c *ConnectionCapture) append(kind string, data []byte, detail string, metadata json.RawMessage, ns int64) error {
	if c.closed {
		return fault("invalid_state", "ConnectionCapture", fmt.Errorf("capture closed"))
	}
	if c.failure != nil {
		return c.failure
	}
	if ns < c.lastNS || len(data) > 65536 || len(detail) > 1024 {
		return fault("invalid_input", "ConnectionCapture", fmt.Errorf("clock regression or oversized input"))
	}
	row := captureRow{1, c.id, c.seq + 1, kind, ns, c.anchorMS + (ns-c.anchorNS)/1000000, data, detail, metadata}
	b, err := json.Marshal(row)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	limit := int64(64<<20) - 8192
	if kind == "closed" || kind == "error" {
		limit = 64 << 20
	}
	if c.size+int64(len(b)) > limit {
		return fault("resource_limit", "ConnectionCapture", fmt.Errorf("64 MiB capture limit; stop transport"))
	}
	n, err := c.file.Write(b)
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = c.file.Sync()
	}
	if err != nil {
		c.failure = fault("io_failure", "ConnectionCapture", err)
		return c.failure
	}
	c.size += int64(n)
	c.seq++
	c.lastNS = ns
	if len(row.Bytes) > 256 {
		row.Bytes = append([]byte(nil), row.Bytes[:256]...)
		row.Detail = "preview shortened; full bytes on disk"
	} else {
		row.Bytes = append([]byte(nil), row.Bytes...)
	}
	c.recent = append(c.recent, row)
	if len(c.recent) > 32 {
		c.recent = c.recent[len(c.recent)-32:]
	}
	return nil
}

// Receive transfers owned arbitrary byte chunks, preserving their exact order.
func (c *ConnectionCapture) Receive(data []byte, ns int64) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("Receive", &err)
	if len(data) == 0 {
		return fault("invalid_input", "Receive", fmt.Errorf("empty chunk"))
	}
	if err = c.append("received", append([]byte(nil), data...), "", nil, ns); err == nil {
		c.received += int64(len(data))
	}
	return
}

// IdentityRequest is the only M5 command source. OBDLink FRPM E sections 6/8.4:
// STI reads adapter firmware identity; CR terminates it. Never auto-send/replay.
func (c *ConnectionCapture) IdentityRequest(ns int64) (_ []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("IdentityRequest", &err)
	if c.used {
		return nil, fault("invalid_state", "IdentityRequest", fmt.Errorf("already attempted in this generation"))
	}
	data := []byte("STI\r")
	if err = c.append("write_requested", data, "adapter firmware identity only", nil, ns); err != nil {
		return nil, err
	}
	c.used = true
	c.pending = append([]byte(nil), data...)
	c.requested = 4
	return append([]byte(nil), data...), nil
}

// Accepted records actual OutputStream acceptance, not adapter delivery.
func (c *ConnectionCapture) Accepted(data []byte, ns int64) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("Accepted", &err)
	if len(data) == 0 || len(data) > len(c.pending) || !bytes.Equal(data, c.pending[:len(data)]) {
		return fault("invalid_input", "Accepted", fmt.Errorf("not the pending Go-owned prefix"))
	}
	if err = c.append("write_accepted", append([]byte(nil), data...), "stream accepted; adapter delivery unproven", nil, ns); err == nil {
		c.pending = c.pending[len(data):]
		c.accepted += int64(len(data))
	}
	return
}

// Event records native facts without inferring vehicle/module health.
func (c *ConnectionCapture) Event(kind, detail string, ns int64) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("ConnectionEvent", &err)
	switch kind {
	case "streams_ready", "foreground", "background", "inactive", "error", "identity_timeout":
	default:
		return fault("invalid_input", "ConnectionEvent", fmt.Errorf("unknown event"))
	}
	return c.append(kind, nil, detail, nil, ns)
}

// Close is repeatable and terminal; unfinished requested bytes are explicit.
func (c *ConnectionCapture) Close(reason string, ns int64) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("CloseConnection", &err)
	if c.closed {
		return c.failure
	}
	if ns < c.lastNS {
		ns = c.lastNS
	}
	if len(reason) > 512 {
		reason = reason[:512]
	}
	err = c.append("closed", nil, fmt.Sprintf("%s; requested=%d accepted=%d unsent=%d; adapter delivery unproven", reason, c.requested, c.accepted, len(c.pending)), nil, ns)
	c.closed = true
	c.pending = nil
	e := c.file.Close()
	if err == nil {
		err = e
	}
	c.failure = err
	return
}
func (c *ConnectionCapture) Path() string { return c.path }
func (c *ConnectionCapture) Snapshot() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	failure := ""
	if c.failure != nil {
		failure = c.failure.Error()
	}
	b, _ := json.Marshal(map[string]any{"capture_version": 1, "connection_id": c.id, "received_bytes": c.received, "accepted_bytes": c.accepted, "requested_bytes": c.requested, "closed": c.closed, "error": failure, "recent": c.recent})
	return string(b)
}
