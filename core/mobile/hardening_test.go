package mobile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"vehicleflightrecorder.local/core/telemetry"
)

func newTestController(t *testing.T, root string) *Controller {
	t.Helper()
	pack, _ := filepath.Abs("../../vehicle-packs/alfa-giulia-qv-2017")
	c, err := NewController(pack, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}
func code(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "VFR|"+expected+"|") {
		t.Fatalf("expected %s, got %v", expected, err)
	}
}
func TestCompatibilityAndTerminalClose(t *testing.T) {
	if err := CheckBridgeVersion(BridgeVersion()); err != nil {
		t.Fatal(err)
	}
	code(t, CheckBridgeVersion(BridgeVersion()+1), "unsupported_version")
	if !json.Valid([]byte(BridgeContractJSON())) {
		t.Fatal("invalid contract")
	}
	c := newTestController(t, t.TempDir())
	code(t, c.Start("unknown"), "invalid_input")
	if err := c.Start("healthy_drive"); err != nil {
		t.Fatal(err)
	}
	id := c.core.Session().SessionID
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	code(t, c.Start("healthy_drive"), "invalid_state")
	code(t, c.Advance(100), "invalid_state")
	_, err := c.Snapshot()
	code(t, err, "invalid_state")
	select {
	case <-c.producerDone:
	default:
		t.Fatal("producer survived close")
	}
	metadata, err := readSession(filepath.Join(c.root, id))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.State != "recording" {
		t.Fatal("close silently finalized evidence")
	}
	reopened := newTestController(t, c.root)
	text, err := reopened.SessionsJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, `"state":"recording"`) {
		t.Fatal(text)
	}
	_, err = reopened.ExportPath(id)
	code(t, err, "invalid_state")
}
func TestStaleGenerationCannotAdvanceNewSession(t *testing.T) {
	c := newTestController(t, t.TempDir())
	_ = c.Start("healthy_drive")
	old := c.core.Session().SessionID
	_ = c.Stop()
	_ = c.Start("healthy_drive")
	before := c.clock
	code(t, c.AdvanceSession(old, 2500), "cancelled")
	if c.clock != before {
		t.Fatal("old tick advanced new session")
	}
	if err := c.AdvanceSession(c.core.Session().SessionID, 2500); err != nil {
		t.Fatal(err)
	}
}
func TestPagedHistoryAndInputBounds(t *testing.T) {
	c := newTestController(t, t.TempDir())
	for i := 0; i < 55; i++ {
		if err := c.Start("healthy_drive"); err != nil {
			t.Fatal(err)
		}
		if err := c.Stop(); err != nil {
			t.Fatal(err)
		}
	}
	var first, second struct {
		Sessions []json.RawMessage `json:"sessions"`
		HasMore  bool              `json:"has_more"`
		Next     int64             `json:"next_offset"`
	}
	raw, err := c.SessionsPageJSON(0)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal([]byte(raw), &first)
	raw, err = c.SessionsPageJSON(first.Next)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal([]byte(raw), &second)
	if len(first.Sessions) != 50 || !first.HasMore || len(second.Sessions) != 5 || second.HasMore {
		t.Fatal("incorrect paging")
	}
	_, err = c.SessionsPageJSON(-1)
	code(t, err, "invalid_input")
	_, err = c.ExportPath("../escape")
	code(t, err, "invalid_input")
	_, err = c.Mark(strings.Repeat("a", 257))
	code(t, err, "invalid_input")
	for i := 55; i <= maxHistoryEntries; i++ {
		_ = os.WriteFile(filepath.Join(c.root, fmt.Sprint(i)), nil, 0600)
	}
	_, err = c.SessionsPageJSON(0)
	code(t, err, "resource_limit")
}
func TestMalformedOversizedAndEscapingEvidence(t *testing.T) {
	for name, contents := range map[string][]byte{"truncated": []byte(`{"x":1}`), "malformed": []byte("{bad}\n"), "oversized": []byte(strings.Repeat("a", maxJSONBytes+2))} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			_ = os.WriteFile(filepath.Join(root, "events.jsonl"), contents, 0600)
			err := scanJSON(root, "events.jsonl", func(_ int64, _ json.RawMessage) error { return nil })
			if err == nil {
				t.Fatal("accepted invalid evidence")
			}
		})
	}
	root := t.TempDir()
	external := filepath.Join(t.TempDir(), "outside")
	_ = os.WriteFile(external, []byte("{}\n"), 0600)
	_ = os.Symlink(external, filepath.Join(root, "session.json"))
	_, err := readSession(root)
	code(t, err, "corrupt_data")
	var target map[string]any
	code(t, decodeDocument([]byte(`{} {}`), &target), "corrupt_data")
	code(t, decodeDocument([]byte(strings.Repeat(" ", maxJSONBytes+1)), &target), "resource_limit")
}
func TestFinalizeFailurePoisonsFacadeAndPreservesEvidence(t *testing.T) {
	c := newTestController(t, t.TempDir())
	if err := c.Start("healthy_drive"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(c.root, c.core.Session().SessionID)
	// Replace the metadata destination with a directory to force rename failure.
	if err := os.Rename(filepath.Join(dir, "session.json"), filepath.Join(dir, "original-session.json")); err != nil {
		t.Fatal(err)
	}
	_ = os.Mkdir(filepath.Join(dir, "session.json"), 0700)
	code(t, c.Stop(), "io_failure")
	snapshot, err := c.Snapshot()
	if err != nil || !strings.Contains(snapshot, `"state":"failed"`) {
		t.Fatalf("%s %v", snapshot, err)
	}
	code(t, c.Start("healthy_drive"), "invalid_state")
	if _, err := os.Stat(filepath.Join(dir, "original-session.json")); err != nil {
		t.Fatal(err)
	}
}
func TestConcurrentCloseAndRepeatedLogicalDrives(t *testing.T) {
	c := newTestController(t, t.TempDir())
	start := time.Now()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for drive := 0; drive < 20; drive++ {
		if err := c.Start("healthy_drive"); err != nil {
			t.Fatal(err)
		}
		for tick := 0; tick < 72; tick++ {
			if err := c.Advance(10000); err != nil {
				t.Fatal(err)
			}
		}
		select {
		case <-c.producerDone:
		default:
			t.Fatal("producer not joined")
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	t.Logf("20 complete 12-minute logical healthy drives (4 logical hours), wall=%s; heap before=%d after=%d bytes", time.Since(start), before.HeapAlloc, after.HeapAlloc)
	_ = c.Start("healthy_drive")
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = c.Snapshot()
			}
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); _ = c.Close() }()
	wg.Wait()
	code(t, c.Advance(100), "invalid_state")
}

func TestManualBudgetDoesNotAppendOrPoisonRecording(t *testing.T) {
	c := newTestController(t, t.TempDir())
	_ = c.Start("healthy_drive")
	c.markCount = maxManualMarks
	before := c.core.Session().EventCount
	_, err := c.Mark("over budget")
	code(t, err, "resource_limit")
	if c.core.Session().EventCount != before || c.core.State() != "recording" {
		t.Fatal("budget altered recording")
	}
	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestMetadataBudgetAndUnknownObservationFields(t *testing.T) {
	c := newTestController(t, t.TempDir())
	_ = c.Start("healthy_drive")
	_ = c.Stop()
	original := c.core.Session()
	for i := 0; i < 12; i++ {
		value := original
		value.SessionID = fmt.Sprintf("import-%d", i)
		value.ProfileID = strings.Repeat("x", 800000)
		path := filepath.Join(c.root, value.SessionID)
		_ = os.Mkdir(path, 0700)
		raw, _ := json.Marshal(value)
		_ = os.WriteFile(filepath.Join(path, "session.json"), raw, 0600)
	}
	_, err := c.SessionsPageJSON(0)
	code(t, err, "resource_limit")
	var observation telemetry.Observation
	code(t, decodeDocument([]byte(`{"schema_version":1,"timestamp_unix_ms":1,"session_id":"test","source":"mock","signal_id":"s","value":0,"unit":"V","quality":"valid","unexpected":true}`), &observation), "corrupt_data")
}
