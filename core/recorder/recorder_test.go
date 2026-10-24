package recorder

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vehicleflightrecorder.local/core/incidents"
	"vehicleflightrecorder.local/core/telemetry"
	"vehicleflightrecorder.local/core/vehicle"
)

func observation(timestamp int64) telemetry.Record {
	return telemetry.Record{Observation: &telemetry.Observation{SchemaVersion: 1, TimestampUnixMS: timestamp, SessionID: "s", SignalID: "rpm", Source: "test", Value: 800, Unit: "rpm", Quality: "valid"}}
}

func metadata() telemetry.Session {
	return telemetry.Session{SchemaVersion: 1, SessionID: "s", Source: "test", ProfileID: "generic", VehiclePackVersion: "0.1.0", StartTimeUnixMS: 1000, State: "recording"}
}

func TestBufferInclusiveCutoffOrderAndOwnership(t *testing.T) {
	b, err := NewBuffer(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	original := observation(1000)
	if err := b.Add(original); err != nil {
		t.Fatal(err)
	}
	original.Observation.Value = -1
	for _, timestamp := range []int64{1500, 2000} {
		if err := b.Add(observation(timestamp)); err != nil {
			t.Fatal(err)
		}
	}
	if got := b.Snapshot(); len(got) != 3 || got[0].Observation.Value != 800 {
		t.Fatal("cutoff/ownership failed", got)
	}
	if err := b.Add(observation(2001)); err != nil {
		t.Fatal(err)
	}
	got := b.Snapshot()
	if len(got) != 3 || got[0].Timestamp() != 1500 {
		t.Fatal("expired cutoff not removed", got)
	}
	got[0].Observation.Value = 1
	if b.Snapshot()[0].Observation.Value != 800 {
		t.Fatal("snapshot aliases buffer")
	}
	if b.Add(observation(1999)) == nil {
		t.Fatal("backwards timestamp accepted")
	}
	if err := b.Advance(4000); err != nil {
		t.Fatal(err)
	}
	if len(b.Snapshot()) != 0 {
		t.Fatal("idle advance did not expire samples")
	}
	if _, err := NewBuffer(0); err == nil {
		t.Fatal("zero window accepted")
	}
}

func TestBufferMixedSameTimeAndEventCopy(t *testing.T) {
	b, _ := NewBuffer(time.Minute)
	e := telemetry.Event{SchemaVersion: 1, TimestampUnixMS: 1000, SessionID: "s", Source: "test", EventType: "phase", Severity: "info", Attributes: json.RawMessage(`{"phase":"idle"}`)}
	if err := b.Add(telemetry.Record{Event: &e}); err != nil {
		t.Fatal(err)
	}
	if err := b.Add(observation(1000)); err != nil {
		t.Fatal(err)
	}
	e.Attributes[2] = 'X'
	snapshot := b.Snapshot()
	if snapshot[0].Event == nil || snapshot[1].Observation == nil || string(snapshot[0].Event.Attributes) != `{"phase":"idle"}` {
		t.Fatal("mixed ordering/event ownership failed")
	}
	snapshot[0].Event.Attributes[2] = 'Y'
	if string(b.Snapshot()[0].Event.Attributes) != `{"phase":"idle"}` {
		t.Fatal("event snapshot aliases buffer")
	}
}

func TestLifecycleAndIncompleteClose(t *testing.T) {
	r, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if r.Append(observation(1000)) == nil || r.Stop(2000) == nil {
		t.Fatal("idle lifecycle operations accepted")
	}
	dir := filepath.Join(t.TempDir(), "session")
	if err := r.Start(dir, metadata()); err != nil {
		t.Fatal(err)
	}
	if r.Start(dir, metadata()) == nil {
		t.Fatal("double start accepted")
	}
	if err := r.Append(observation(1000)); err != nil {
		t.Fatal(err)
	}
	if r.Stop(999) == nil {
		t.Fatal("end before start accepted")
	}
	wrong := observation(2000)
	wrong.Observation.SessionID = "other"
	if r.Append(wrong) == nil {
		t.Fatal("foreign session accepted")
	}
	if err := r.Stop(2000); err != nil {
		t.Fatal(err)
	}
	if r.State() != "complete" || r.Session().ObservationCount != 1 || r.Session().EventCount != 2 {
		t.Fatal("wrong completed session", r.Session())
	}
	if r.Append(observation(2000)) == nil || r.Stop(2000) == nil {
		t.Fatal("completed recorder accepted writes/stop")
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	var stored telemetry.Session
	data, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.State != "complete" || stored.EventCount != 2 {
		t.Fatal("final metadata not committed")
	}
	other, _ := New(time.Minute)
	incomplete := filepath.Join(t.TempDir(), "interrupted")
	if err := other.Start(incomplete, metadata()); err != nil {
		t.Fatal(err)
	}
	if err := other.Append(observation(1000)); err != nil {
		t.Fatal(err)
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(incomplete, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.State != "recording" || stored.EndTimeUnixMS != nil {
		t.Fatal("unfinalized recording falsely complete")
	}
}

func TestEarlyMockStopAndInvalidEndDoNotCorruptLifecycle(t *testing.T) {
	r, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	s := metadata()
	s.Source = "mock"
	s.Mock = &telemetry.MockConfig{Scenario: "healthy_drive", Seed: 42, DurationMS: 120000, StepMS: 100, GeneratorVersion: "1"}
	if err := r.Start(filepath.Join(t.TempDir(), "session"), s); err != nil {
		t.Fatal(err)
	}
	if r.Stop(122000) == nil || r.State() != "recording" {
		t.Fatal("invalid end should reject without recording a stop")
	}
	if err := r.Stop(3000); err != nil {
		t.Fatal(err)
	}
	if r.Session().EventCount != 2 || *r.Session().EndTimeUnixMS != 3000 || r.Session().Mock.DurationMS != 120000 {
		t.Fatal("early stop lost actual/planned duration or lifecycle events")
	}
}

type failingSink struct{ appendErr, finalizeErr error }

func (s failingSink) Append(telemetry.Record) error    { return s.appendErr }
func (s failingSink) Finalize(telemetry.Session) error { return s.finalizeErr }
func (s failingSink) Close() error                     { return nil }

func TestSinkFailurePoisonsSession(t *testing.T) {
	failure := errors.New("disk failure")
	for _, finalization := range []bool{false, true} {
		r, _ := New(time.Minute)
		if err := r.Start(filepath.Join(t.TempDir(), "session"), metadata()); err != nil {
			t.Fatal(err)
		}
		if err := r.sink.Close(); err != nil {
			t.Fatal(err)
		}
		if finalization {
			r.sink = failingSink{finalizeErr: failure}
			if !errors.Is(r.Stop(2000), failure) {
				t.Fatal("lost finalize error")
			}
		} else {
			r.sink = failingSink{appendErr: failure}
			if !errors.Is(r.Append(observation(2000)), failure) {
				t.Fatal("lost append error")
			}
			if len(r.Snapshot()) != 1 {
				t.Fatal("failed write entered rolling buffer")
			}
		}
		if r.State() != "failed" || r.Session().State != "recording" || r.Stop(2000) == nil {
			t.Fatal("failed session allowed finalization")
		}
	}
}

func TestManualIncidentLifecycle(t *testing.T) {
	p, err := vehicle.Load("../../vehicle-packs/alfa-giulia-qv-2017")
	if err != nil {
		t.Fatal(err)
	}
	c, err := vehicle.LoadIncidentConfig("../../vehicle-packs/alfa-giulia-qv-2017", p)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ConfigureIncidents(c); err != nil {
		t.Fatal(err)
	}
	if _, err := r.MarkIncident("before", 1000); err == nil {
		t.Fatal("mark before start")
	}
	s := telemetry.Session{SchemaVersion: 1, SessionID: "manual", Source: "fixture", ProfileID: p.Manifest.ProfileID, VehiclePackVersion: p.Manifest.PackVersion, StartTimeUnixMS: 1000, State: "recording"}
	dir := filepath.Join(t.TempDir(), "session")
	if err := r.Start(dir, s); err != nil {
		t.Fatal(err)
	}
	id, err := r.MarkIncident("manual", 1000)
	if err != nil || id != "manual.incident-000001" {
		t.Fatal(id, err)
	}
	if _, err := r.MarkIncident("", 1000); err == nil {
		t.Fatal("empty label accepted")
	}
	if err := r.ConfigureIncidents(c); err == nil {
		t.Fatal("changed active configuration")
	}
	if err := r.Stop(2000); err != nil {
		t.Fatal(err)
	}
	if r.Session().IncidentCount != 1 || r.Incidents()[0].Status != "partial" {
		t.Fatal(r.Session(), r.Incidents())
	}
	if _, err := r.MarkIncident("after", 2000); err == nil {
		t.Fatal("mark after stop")
	}
}

type failingIncidentSink struct {
	failingSink
	err error
}

func (s failingIncidentSink) AppendIncident(incidents.Incident) error { return s.err }
func TestIncidentWriteFailurePreservesIncompleteSession(t *testing.T) {
	p, err := vehicle.Load("../../vehicle-packs/alfa-giulia-qv-2017")
	if err != nil {
		t.Fatal(err)
	}
	c, err := vehicle.LoadIncidentConfig("../../vehicle-packs/alfa-giulia-qv-2017", p)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := New(time.Minute)
	if err := r.ConfigureIncidents(c); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "failed")
	if err := r.Start(dir, metadata()); err != nil {
		t.Fatal(err)
	}
	original := r.sink
	defer original.Close()
	failure := errors.New("incident disk failure")
	r.sink = failingIncidentSink{err: failure}
	if _, err := r.MarkIncident("fault", 2000); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if r.State() != "failed" || r.Stop(3000) == nil {
		t.Fatal("incident failure must prevent finalization")
	}
	data, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var session telemetry.Session
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatal(err)
	}
	if session.State != "recording" {
		t.Fatal("incomplete evidence marked complete")
	}
}
