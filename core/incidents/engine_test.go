package incidents

import (
	"encoding/json"
	"reflect"
	"testing"

	"vehicleflightrecorder.local/core/telemetry"
)

func config() Config {
	return Config{Availability: "mock_only", Provenance: "test", PreWindowMS: 60000, PostWindowMS: 120000, MaxSilenceMS: 2000, VoltageSignalID: "voltage", VoltageUnit: "V", LowVoltage: 12, ResetVoltage: 13, DropDelta: 1.5, DropWindowMS: 1000, TimeoutCount: 3, CriticalModules: []string{"TCM", "ECM"}, CorrelationWindowMS: 1000, TransportLoss: true}
}
func observation(at int64, value float64) telemetry.Record {
	return telemetry.Record{Observation: &telemetry.Observation{SchemaVersion: 1, TimestampUnixMS: at, SessionID: "s", Source: "mock", SignalID: "voltage", Value: value, Unit: "V", Quality: "valid"}}
}
func event(at int64, kind, module string, attrs any) telemetry.Record {
	if attrs == nil {
		attrs = map[string]any{}
	}
	data, _ := json.Marshal(attrs)
	return telemetry.Record{Event: &telemetry.Event{SchemaVersion: 1, TimestampUnixMS: at, SessionID: "s", Source: "mock", EventType: kind, Module: module, Severity: "warning", Attributes: data}}
}
func evaluate(t *testing.T, d Detector, r telemetry.Record, want ...string) {
	t.Helper()
	got, err := d.Evaluate(r)
	if err != nil {
		t.Fatal(err)
	}
	types := []string{}
	for _, c := range got {
		types = append(types, c.Type)
	}
	if !reflect.DeepEqual(types, append([]string{}, want...)) {
		t.Fatalf("got %v want %v", types, want)
	}
}
func TestVoltageCrossingResetAndQuality(t *testing.T) {
	d, _ := NewDetector(config())
	evaluate(t, d, observation(1000, 13.7))
	evaluate(t, d, observation(1100, 12.1), "voltage_excursion")
	evaluate(t, d, observation(1200, 11.5))
	evaluate(t, d, observation(1300, 13.7))
	evaluate(t, d, observation(1400, 11.5), "voltage_excursion")
	d, _ = NewDetector(config())
	r := observation(1000, 11)
	r.Observation.Quality = "stale"
	evaluate(t, d, r)
	r.Observation.Quality = "invalid"
	evaluate(t, d, r)
	r.Observation.Unit = "mV"
	r.Observation.Quality = "valid"
	evaluate(t, d, r)
	evaluate(t, d, observation(2000, 12))
	evaluate(t, d, observation(2100, 13.7))
	evaluate(t, d, observation(2200, 12.6))
}
func TestTimeoutRestorationCorrelationAndTransport(t *testing.T) {
	d, _ := NewDetector(config())
	expected := map[string]bool{"expected_response": true}
	at := int64(1000)
	timeout := func(module string, want ...string) {
		evaluate(t, d, event(at, "diagnostic_timeout", module, expected), want...)
		at += 100
	}
	evaluate(t, d, event(at, "diagnostic_timeout", "TCM", map[string]bool{"expected_response": false}))
	timeout("unrelated")
	timeout("TCM")
	timeout("TCM")
	timeout("TCM", "module_timeout")
	timeout("TCM")
	timeout("ECM")
	timeout("ECM")
	timeout("ECM", "module_timeout", "multi_module_loss")
	evaluate(t, d, event(at, "response_restored", "TCM", nil))
	timeout("TCM")
	timeout("TCM")
	timeout("TCM", "module_timeout", "multi_module_loss")
	evaluate(t, d, event(at, "transport_disconnected", "", nil), "transport_disconnect")
	evaluate(t, d, event(at, "transport_disconnected", "", nil))
	for n := 0; n < 4; n++ {
		timeout("TCM")
	}
	evaluate(t, d, observation(at, 11))
	evaluate(t, d, event(at, "transport_connected", "", nil))
	timeout("TCM")
	timeout("TCM")
	timeout("TCM", "module_timeout")
}
func TestDTCAndManual(t *testing.T) {
	d, _ := NewDetector(config())
	attrs := map[string]string{"code": "MOCK", "category": "powertrain"}
	evaluate(t, d, event(1000, "dtc_observed", "ECM", attrs), "new_dtc")
	evaluate(t, d, event(1100, "dtc_observed", "ECM", attrs))
	evaluate(t, d, event(1200, "dtc_removed", "ECM", attrs))
	evaluate(t, d, event(1300, "dtc_observed", "ECM", attrs), "new_dtc")
	evaluate(t, d, event(1400, "manual_incident_marked", "", map[string]string{"label": "mark"}), "manual_marker")
	if _, err := d.Evaluate(event(1500, "manual_incident_marked", "", map[string]string{})); err == nil {
		t.Fatal("empty marker accepted")
	}
}

func process(t *testing.T, e *Engine, r telemetry.Record, buffer []telemetry.Record) Result {
	t.Helper()
	seq := int64(1)
	if r.Sequence() > 0 {
		seq = r.Sequence()
	}
	r = r.WithSequence(seq)
	ref := Reference{Line: seq, RecordSequence: seq, TimestampUnixMS: r.Timestamp()}
	if r.Event != nil {
		ref.File = "events.jsonl"
		ref.EventType = r.Event.EventType
	} else {
		ref.File = "observations.jsonl"
		ref.SignalID = r.Observation.SignalID
	}
	got, err := e.Process(r, ref, buffer)
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range got.Updates {
		if err := i.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	return got
}
func mark(at int64) telemetry.Record {
	return event(at, "manual_incident_marked", "", map[string]string{"label": "manual"})
}
func TestCaptureBoundariesMergeAndLinks(t *testing.T) {
	c := config()
	c.PreWindowMS = 1000
	c.PostWindowMS = 2000
	c.MaxSilenceMS = 10000
	e, _ := New(c, "s", 1000)
	process(t, e, observation(1000, 13.7), nil)
	a := process(t, e, mark(2000).WithSequence(2), []telemetry.Record{observation(1000, 13.7)})
	if len(a.Updates) != 1 || a.Updates[0].WindowStartUnixMS != 1000 || a.Updates[0].WindowEndUnixMS != 4000 {
		t.Fatal(a)
	}
	process(t, e, mark(3000).WithSequence(3), nil)
	if e.Count() != 1 || len(e.Snapshot()[0].Triggers) != 2 || e.Snapshot()[0].WindowEndUnixMS != 5000 {
		t.Fatal(e.Snapshot())
	}
	process(t, e, observation(4999, 13.7), nil)
	if e.Snapshot()[0].Status != "capturing" {
		t.Fatal("early completion")
	}
	process(t, e, observation(5000, 13.7), nil)
	if e.Snapshot()[0].Status != "captured" {
		t.Fatal(e.Snapshot())
	}
	process(t, e, mark(5500).WithSequence(4), []telemetry.Record{observation(4500, 13.7)})
	if e.Count() != 2 || len(e.Snapshot()[0].RelatedIncidentIDs) != 1 {
		t.Fatal("completed overlap must link")
	}
	result, err := e.Finish(6000)
	if err != nil || result.Updates[0].Status != "partial" {
		t.Fatal(result, err)
	}
	copy := e.Snapshot()
	copy[0].Triggers[0].Label = "mutated"
	if e.Snapshot()[0].Triggers[0].Label == "mutated" {
		t.Fatal("snapshot alias")
	}
}
func TestPartialStartSilenceAndDisconnect(t *testing.T) {
	c := config()
	c.PreWindowMS = 1000
	c.PostWindowMS = 2000
	c.MaxSilenceMS = 500
	e, _ := New(c, "s", 1000)
	process(t, e, mark(1000), []telemetry.Record{mark(1000)})
	process(t, e, observation(2000, 13.7), nil)
	result, err := e.Finish(2500)
	if err != nil {
		t.Fatal(err)
	}
	item := result.Updates[0]
	if item.Status != "partial" || !reflect.DeepEqual(item.IncompleteReasons, []string{"pre_window_unavailable", "post_window_unavailable", "coverage_gap"}) {
		t.Fatal(item)
	}
	e, _ = New(c, "s", 1000)
	process(t, e, observation(1000, 13.7), nil)
	process(t, e, event(2000, "transport_disconnected", "", nil), []telemetry.Record{observation(1000, 13.7)})
	process(t, e, event(2500, "transport_connected", "", nil), nil)
	process(t, e, observation(4000, 13.7), nil)
	if e.Snapshot()[0].Status != "partial" || len(e.Snapshot()[0].CoverageGaps) < 1 {
		t.Fatal(e.Snapshot())
	}
}
func TestInvalidConfigurationAndReference(t *testing.T) {
	c := config()
	c.PreWindowMS = -1
	if _, err := New(c, "s", 1000); err == nil {
		t.Fatal("invalid config")
	}
	e, _ := New(config(), "s", 1000)
	if _, err := e.Process(mark(1000), Reference{}, nil); err == nil {
		t.Fatal("invalid reference")
	}
}

func TestTriggerAtExactDeadlineExtendsCapture(t *testing.T) {
	c := config()
	c.PreWindowMS = 0
	c.PostWindowMS = 1000
	c.MaxSilenceMS = 10000
	e, _ := New(c, "s", 1000)
	process(t, e, mark(1000).WithSequence(1), nil)
	result := process(t, e, mark(2000).WithSequence(2), nil)
	if len(result.Finalized) != 0 || e.Count() != 1 || e.Snapshot()[0].WindowEndUnixMS != 3000 {
		t.Fatal(result, e.Snapshot())
	}
	process(t, e, observation(3000, 13.7).WithSequence(3), nil)
	if e.Snapshot()[0].Status != "captured" {
		t.Fatal(e.Snapshot())
	}
}
