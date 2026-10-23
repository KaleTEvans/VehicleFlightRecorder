package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"vehicleflightrecorder.local/core/telemetry"
)

func initial() telemetry.Session {
	return telemetry.Session{SchemaVersion: 1, SessionID: "s", Source: "test", ProfileID: "generic", VehiclePackVersion: "0.1.0", StartTimeUnixMS: 1000, State: "recording"}
}

func sample() telemetry.Record {
	return telemetry.Record{Observation: &telemetry.Observation{SchemaVersion: 1, TimestampUnixMS: 1000, SessionID: "s", SignalID: "rpm", Source: "test", Value: 0, Unit: "rpm", Quality: "valid"}}
}

type brokenWriter struct{ err error }

func (w brokenWriter) Write(p []byte) (int, error) { return 0, w.err }

func TestJSONLErrors(t *testing.T) {
	if !errors.Is(WriteJSONL(brokenWriter{}, sample()), io.ErrShortWrite) {
		t.Fatal("short write lost")
	}
	failure := errors.New("full disk")
	if !errors.Is(WriteJSONL(brokenWriter{failure}, sample()), failure) {
		t.Fatal("write error lost")
	}
	if WriteJSONL(io.Discard, make(chan int)) == nil {
		t.Fatal("unencodable record accepted")
	}
}

func TestSessionRoundTripAndOverwriteProtection(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "session")
	w, err := Create(dir, initial())
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Append(sample()); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(dir, initial()); err == nil {
		t.Fatal("existing directory overwritten")
	}
	complete := initial()
	end := int64(2000)
	complete.State, complete.EndTimeUnixMS, complete.ObservationCount = "complete", &end, 1
	if err := w.Finalize(complete); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"events.jsonl", "incidents.jsonl", "raw/transport.jsonl"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || len(data) != 0 {
			t.Fatalf("expected empty %s: %v", name, err)
		}
	}
	file, err := os.Open(filepath.Join(dir, "observations.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatal("missing observation")
	}
	var o telemetry.Observation
	if err := json.Unmarshal(scanner.Bytes(), &o); err != nil {
		t.Fatal(err)
	}
	if o.Value != 0 || o.SignalID != "rpm" {
		t.Fatal("wrong stored observation")
	}
	if scanner.Scan() || scanner.Err() != nil {
		t.Fatal("unexpected extra/corrupt record")
	}
	data, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored telemetry.Session
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.State != "complete" || stored.ObservationCount != 1 {
		t.Fatal("wrong final metadata")
	}
}

func TestWriteAndFinalizationFailureLeaveIncompleteEvidence(t *testing.T) {
	for _, failAppend := range []bool{true, false} {
		dir := filepath.Join(t.TempDir(), "session")
		w, err := Create(dir, initial())
		if err != nil {
			t.Fatal(err)
		}
		if failAppend {
			if err := w.observations.Close(); err != nil {
				t.Fatal(err)
			}
			if w.Append(sample()) == nil {
				t.Fatal("closed file write reported success")
			}
		} else {
			if err := w.Append(sample()); err != nil {
				t.Fatal(err)
			}
			// Inject failure at final metadata creation, after logs are flushed/closed.
			w.directory = filepath.Join(dir, "absent-parent", "absent")
		}
		complete := initial()
		end := int64(2000)
		complete.State, complete.EndTimeUnixMS, complete.ObservationCount = "complete", &end, 1
		if w.Finalize(complete) == nil {
			t.Fatal("failed writer finalized")
		}
		_ = w.Close() // A deliberately closed descriptor can report a second-close error.
		data, err := os.ReadFile(filepath.Join(dir, "session.json"))
		if err != nil {
			t.Fatal(err)
		}
		var stored telemetry.Session
		if err := json.Unmarshal(data, &stored); err != nil {
			t.Fatal(err)
		}
		if stored.State != "recording" {
			t.Fatal("failed session marked complete")
		}
	}
}

func TestSyncFailureDoesNotCommitCompleteMetadata(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "session")
	w, err := Create(dir, initial())
	if err != nil {
		t.Fatal(err)
	}
	if err := w.events.Close(); err != nil {
		t.Fatal(err)
	}
	complete := initial()
	end := int64(2000)
	complete.State, complete.EndTimeUnixMS = "complete", &end
	if w.Finalize(complete) == nil {
		t.Fatal("failed sync incorrectly reported success")
	}
	data, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored telemetry.Session
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.State != "recording" {
		t.Fatal("failed sync marked session complete")
	}
}
