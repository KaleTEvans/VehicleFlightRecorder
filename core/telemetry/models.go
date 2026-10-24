package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

const SchemaVersion = 1

// Observation is a numeric canonical measurement; missing values are not zeroes.
// Only present, finite values are supported by this first schema implementation.
type Observation struct {
	RecordSequence     int64   `json:"record_sequence,omitempty"`
	SchemaVersion      int     `json:"schema_version"`
	TimestampUnixMS    int64   `json:"timestamp_unix_ms"`
	SessionID          string  `json:"session_id"`
	SignalID           string  `json:"signal_id"`
	Source             string  `json:"source"`
	Value              float64 `json:"value"`
	Unit               string  `json:"unit"`
	Quality            string  `json:"quality"`
	Module             string  `json:"module,omitempty"`
	Bus                string  `json:"bus,omitempty"`
	RawRequest         string  `json:"raw_request,omitempty"`
	RawResponse        string  `json:"raw_response,omitempty"`
	DecoderVersion     string  `json:"decoder_version,omitempty"`
	VehiclePackVersion string  `json:"vehicle_pack_version,omitempty"`
}

func (o Observation) Validate() error {
	if o.RecordSequence < 0 {
		return fmt.Errorf("record_sequence cannot be negative")
	}
	if err := validateEnvelope(o.SchemaVersion, o.TimestampUnixMS, o.SessionID, o.Source); err != nil {
		return err
	}
	if strings.TrimSpace(o.SignalID) == "" || strings.TrimSpace(o.Unit) == "" {
		return fmt.Errorf("signal_id and unit are required")
	}
	if math.IsNaN(o.Value) || math.IsInf(o.Value, 0) {
		return fmt.Errorf("observation value must be finite")
	}
	switch o.Quality {
	case "valid", "stale", "invalid":
	default:
		return fmt.Errorf("unsupported quality %q", o.Quality)
	}
	return nil
}

func (o *Observation) UnmarshalJSON(data []byte) error {
	type plain Observation
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if value, ok := fields["value"]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return fmt.Errorf("observation value is required")
	}
	*o = Observation(decoded)
	return o.Validate()
}

type Event struct {
	RecordSequence  int64  `json:"record_sequence,omitempty"`
	SchemaVersion   int    `json:"schema_version"`
	TimestampUnixMS int64  `json:"timestamp_unix_ms"`
	SessionID       string `json:"session_id"`
	Source          string `json:"source"`
	EventType       string `json:"event_type"`
	Severity        string `json:"severity"`
	Module          string `json:"module,omitempty"`
	// JSON object bytes preserve arbitrary future attributes without sharing maps.
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

func (e Event) Validate() error {
	if e.RecordSequence < 0 {
		return fmt.Errorf("record_sequence cannot be negative")
	}
	if err := validateEnvelope(e.SchemaVersion, e.TimestampUnixMS, e.SessionID, e.Source); err != nil {
		return err
	}
	if strings.TrimSpace(e.EventType) == "" {
		return fmt.Errorf("event_type is required")
	}
	switch e.Severity {
	case "info", "warning", "error":
	default:
		return fmt.Errorf("unsupported severity %q", e.Severity)
	}
	if len(e.Attributes) != 0 {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(e.Attributes, &object); err != nil || object == nil {
			return fmt.Errorf("event attributes must be a JSON object")
		}
	}
	return nil
}

func validateEnvelope(version int, timestamp int64, session, source string) error {
	if version != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", version)
	}
	if timestamp <= 0 {
		return fmt.Errorf("timestamp_unix_ms must be positive")
	}
	if strings.TrimSpace(session) == "" || strings.TrimSpace(source) == "" {
		return fmt.Errorf("session_id and source are required")
	}
	return nil
}

// Record is the in-process union. Files contain the observation/event directly.
type Record struct {
	Observation *Observation
	Event       *Event
}

func (r Record) Validate() error {
	if (r.Observation == nil) == (r.Event == nil) {
		return fmt.Errorf("record must contain exactly one observation or event")
	}
	if r.Observation != nil {
		return r.Observation.Validate()
	}
	return r.Event.Validate()
}

func (r Record) Timestamp() int64 {
	if r.Observation != nil {
		return r.Observation.TimestampUnixMS
	}
	return r.Event.TimestampUnixMS
}

func (r Record) SessionID() string {
	if r.Observation != nil {
		return r.Observation.SessionID
	}
	return r.Event.SessionID
}

func (r Record) Source() string {
	if r.Observation != nil {
		return r.Observation.Source
	}
	return r.Event.Source
}

func (r Record) Clone() Record {
	if r.Observation != nil {
		o := *r.Observation
		r.Observation = &o
	}
	if r.Event != nil {
		e := *r.Event
		e.Attributes = bytes.Clone(e.Attributes)
		r.Event = &e
	}
	return r
}

func (r Record) Sequence() int64 {
	if r.Observation != nil {
		return r.Observation.RecordSequence
	}
	return r.Event.RecordSequence
}
func (r Record) WithSequence(sequence int64) Record {
	r = r.Clone()
	if r.Observation != nil {
		r.Observation.RecordSequence = sequence
	} else {
		r.Event.RecordSequence = sequence
	}
	return r
}

type CaptureConfig struct {
	PreWindowMS  int64 `json:"pre_window_ms"`
	PostWindowMS int64 `json:"post_window_ms"`
	MaxSilenceMS int64 `json:"max_silence_ms"`
}

type MockConfig struct {
	Scenario         string `json:"scenario"`
	Seed             int64  `json:"seed"`
	DurationMS       int64  `json:"duration_ms"`
	StepMS           int64  `json:"step_ms"`
	GeneratorVersion string `json:"generator_version"`
}

type Session struct {
	IncidentCapture    *CaptureConfig `json:"incident_capture,omitempty"`
	SchemaVersion      int            `json:"schema_version"`
	SessionID          string         `json:"session_id"`
	Source             string         `json:"source"`
	ProfileID          string         `json:"profile_id"`
	VehiclePackVersion string         `json:"vehicle_pack_version"`
	StartTimeUnixMS    int64          `json:"start_time_unix_ms"`
	EndTimeUnixMS      *int64         `json:"end_time_unix_ms"`
	State              string         `json:"state"`
	ObservationCount   int64          `json:"observation_count"`
	EventCount         int64          `json:"event_count"`
	IncidentCount      int64          `json:"incident_count"`
	Mock               *MockConfig    `json:"mock,omitempty"`
}

func (s Session) Validate() error {
	if err := validateEnvelope(s.SchemaVersion, s.StartTimeUnixMS, s.SessionID, s.Source); err != nil {
		return err
	}
	if strings.TrimSpace(s.ProfileID) == "" || strings.TrimSpace(s.VehiclePackVersion) == "" {
		return fmt.Errorf("profile_id and vehicle_pack_version are required")
	}
	if s.ObservationCount < 0 || s.EventCount < 0 || s.IncidentCount < 0 || (s.IncidentCount > 0 && s.IncidentCapture == nil) {
		return fmt.Errorf("invalid counts or incident capture metadata")
	}
	if c := s.IncidentCapture; c != nil && (c.PreWindowMS < 0 || c.PostWindowMS < 0 || c.PreWindowMS > 86400000 || c.PostWindowMS > 86400000 || c.MaxSilenceMS <= 0 || c.MaxSilenceMS > 86400000) {
		return fmt.Errorf("invalid incident capture metadata")
	}
	switch s.State {
	case "recording":
		if s.EndTimeUnixMS != nil {
			return fmt.Errorf("recording session must not have end time")
		}
	case "complete":
		if s.EndTimeUnixMS == nil || *s.EndTimeUnixMS < s.StartTimeUnixMS {
			return fmt.Errorf("complete session requires an end time at or after start")
		}
	default:
		return fmt.Errorf("unsupported session state %q", s.State)
	}
	if s.Source == "mock" && s.Mock == nil {
		return fmt.Errorf("mock source requires reproducibility metadata")
	}
	if s.Mock != nil {
		if !ValidScenario(s.Mock.Scenario) || s.Mock.DurationMS < 120000 || s.Mock.DurationMS > 86400000 || s.Mock.DurationMS%1000 != 0 || s.Mock.StepMS != 100 || (s.Mock.GeneratorVersion != "1" && s.Mock.GeneratorVersion != "2") || (s.Mock.Scenario != "healthy_drive" && s.Mock.GeneratorVersion != "2") {
			return fmt.Errorf("invalid mock reproducibility metadata")
		}
		if s.StartTimeUnixMS > math.MaxInt64-s.Mock.DurationMS {
			return fmt.Errorf("mock end time overflows timestamp")
		}
		if s.EndTimeUnixMS != nil && *s.EndTimeUnixMS > s.StartTimeUnixMS+s.Mock.DurationMS {
			return fmt.Errorf("session end exceeds configured mock duration")
		}
	}
	return nil
}

func (s Session) Clone() Session {
	if s.IncidentCapture != nil {
		config := *s.IncidentCapture
		s.IncidentCapture = &config
	}
	if s.EndTimeUnixMS != nil {
		end := *s.EndTimeUnixMS
		s.EndTimeUnixMS = &end
	}
	if s.Mock != nil {
		config := *s.Mock
		s.Mock = &config
	}
	return s
}

func ValidScenario(name string) bool {
	switch name {
	case "healthy_drive", "voltage_tcm_dropout", "tcm_dropout", "mechanical_fault", "adapter_disconnect", "manual_mark":
		return true
	}
	return false
}
