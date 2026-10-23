package telemetry

import (
	"encoding/json"
	"math"
	"testing"
)

func TestObservationValidationAndRoundTrip(t *testing.T) {
	valid := Observation{SchemaVersion: 1, TimestampUnixMS: 1000, SessionID: "session", SignalID: "powertrain.engine.rpm", Source: "mock", Unit: "rpm", Quality: "valid"}
	encoded, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Observation
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != valid {
		t.Fatalf("round trip changed observation: %+v", decoded)
	}
	for _, change := range []func(*Observation){
		func(o *Observation) { o.SchemaVersion = 2 }, func(o *Observation) { o.TimestampUnixMS = 0 },
		func(o *Observation) { o.Source = "" }, func(o *Observation) { o.SessionID = " " },
		func(o *Observation) { o.Unit = "" }, func(o *Observation) { o.SignalID = "" },
		func(o *Observation) { o.Quality = "guess" }, func(o *Observation) { o.Value = math.NaN() },
		func(o *Observation) { o.Value = math.Inf(1) },
	} {
		invalid := valid
		change(&invalid)
		if invalid.Validate() == nil {
			t.Fatalf("accepted invalid observation: %+v", invalid)
		}
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	for _, missing := range []any{nil, "absent"} {
		object["value"] = missing
		if missing == "absent" {
			delete(object, "value")
		}
		data, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		if json.Unmarshal(data, &decoded) == nil {
			t.Fatal("accepted missing/null value as zero")
		}
	}
}

func TestEventAndRecordOwnership(t *testing.T) {
	event := Event{SchemaVersion: 1, TimestampUnixMS: 1000, SessionID: "s", Source: "mock", EventType: "phase", Severity: "info", Attributes: json.RawMessage(`{"phase":"idle","details":{"synthetic":true}}`)}
	record := Record{Event: &event}
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	clone := record.Clone()
	clone.Event.Attributes[2] = 'X'
	if string(event.Attributes) != `{"phase":"idle","details":{"synthetic":true}}` {
		t.Fatal("clone shares attribute bytes")
	}
	for _, attrs := range []string{`null`, `[]`, `"text"`, `{"broken"`} {
		event.Attributes = json.RawMessage(attrs)
		if event.Validate() == nil {
			t.Fatalf("accepted attributes %s", attrs)
		}
	}
	if (Record{}).Validate() == nil {
		t.Fatal("accepted empty record")
	}
	if (Record{Event: &event, Observation: &Observation{}}).Validate() == nil {
		t.Fatal("accepted ambiguous record")
	}
}

func TestSessionValidationAndCopy(t *testing.T) {
	session := Session{SchemaVersion: 1, SessionID: "s", Source: "mock", ProfileID: "p", VehiclePackVersion: "0.1.0", StartTimeUnixMS: 1000, State: "recording", Mock: &MockConfig{Scenario: "healthy_drive", Seed: 42, DurationMS: 120000, StepMS: 100, GeneratorVersion: "1"}}
	if err := session.Validate(); err != nil {
		t.Fatal(err)
	}
	copy := session.Clone()
	copy.Mock.Seed = 100
	if session.Mock.Seed != 42 {
		t.Fatal("session exposes mutable configuration")
	}
	session.State = "complete"
	if session.Validate() == nil {
		t.Fatal("complete session missing end time accepted")
	}
	end := int64(121000)
	session.EndTimeUnixMS = &end
	if err := session.Validate(); err != nil {
		t.Fatal(err)
	}
	session.IncidentCount = 1
	if session.Validate() == nil {
		t.Fatal("Milestone 1 accepted incident count")
	}
}
