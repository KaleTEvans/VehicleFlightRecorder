package recorder

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"vehicleflightrecorder.local/core/incidents"
	"vehicleflightrecorder.local/core/storage"
	"vehicleflightrecorder.local/core/telemetry"
)

type sink interface {
	Append(telemetry.Record) error
	Finalize(telemetry.Session) error
	Close() error
}

// Recorder has one caller/owner. Native concurrency is deliberately not implied.
// A failed append/finalization poisons the session; Close preserves incomplete data.
type Recorder struct {
	incidentConfig *incidents.Config
	engine         *incidents.Engine
	buffer         *Buffer
	sink           sink
	session        telemetry.Session
	state          string
}

func New(window time.Duration) (*Recorder, error) {
	buffer, err := NewBuffer(window)
	if err != nil {
		return nil, err
	}
	return &Recorder{buffer: buffer, state: "idle"}, nil
}

func (r *Recorder) Start(directory string, session telemetry.Session) error {
	if r.state != "idle" {
		return fmt.Errorf("start requires idle recorder, got %s", r.state)
	}
	if c := r.incidentConfig; c != nil {
		engine, err := incidents.New(*c, session.SessionID, session.StartTimeUnixMS)
		if err != nil {
			return err
		}
		r.engine = engine
		session = session.Clone()
		session.IncidentCapture = &telemetry.CaptureConfig{PreWindowMS: c.PreWindowMS, PostWindowMS: c.PostWindowMS, MaxSilenceMS: c.MaxSilenceMS}
	}
	writer, err := storage.Create(directory, session)
	if err != nil {
		return err
	}
	r.sink, r.session, r.state = writer, session.Clone(), "recording"
	if err := r.Append(r.lifecycle("recording_started", session.StartTimeUnixMS)); err != nil {
		return errors.Join(err, r.Close())
	}
	return nil
}

func (r *Recorder) Append(record telemetry.Record) error {
	record, ref, err := r.appendRaw(record)
	if err != nil {
		return err
	}
	if r.engine == nil {
		return nil
	}
	// Engine reads this owned buffer synchronously and retains no sample references.
	result, err := r.engine.Process(record, ref, r.buffer.records)
	if err != nil {
		r.state = "failed"
		return err
	}
	return r.persistIncidents(result, record.Timestamp())
}

func (r *Recorder) appendRaw(record telemetry.Record) (telemetry.Record, incidents.Reference, error) {
	if r.state != "recording" {
		return record, incidents.Reference{}, fmt.Errorf("append requires recording state, got %s", r.state)
	}
	if err := r.buffer.check(record); err != nil {
		return record, incidents.Reference{}, err
	}
	if record.SessionID() != r.session.SessionID || record.Source() != r.session.Source || record.Timestamp() < r.session.StartTimeUnixMS {
		return record, incidents.Reference{}, fmt.Errorf("record does not belong to active session/source/time range")
	}
	record = record.WithSequence(r.session.ObservationCount + r.session.EventCount + 1)
	if err := r.sink.Append(record); err != nil {
		r.state = "failed"
		return record, incidents.Reference{}, err
	}
	if err := r.buffer.Add(record); err != nil {
		r.state = "failed"
		return record, incidents.Reference{}, err
	}
	if record.Observation != nil {
		r.session.ObservationCount++
	} else {
		r.session.EventCount++
	}
	ref := incidents.Reference{RecordSequence: record.Sequence(), TimestampUnixMS: record.Timestamp()}
	if o := record.Observation; o != nil {
		ref.File = "observations.jsonl"
		ref.Line = r.session.ObservationCount
		ref.SignalID = o.SignalID
	} else {
		ref.File = "events.jsonl"
		ref.Line = r.session.EventCount
		ref.EventType = record.Event.EventType
	}
	return record, ref, nil
}

func (r *Recorder) Stop(endTimeUnixMS int64) error {
	if r.state != "recording" {
		return fmt.Errorf("stop requires recording state, got %s", r.state)
	}
	if endTimeUnixMS < r.session.StartTimeUnixMS || endTimeUnixMS < r.buffer.latest {
		return fmt.Errorf("end time precedes session data")
	}
	complete := r.session.Clone()
	complete.State, complete.EndTimeUnixMS = "complete", &endTimeUnixMS
	if err := complete.Validate(); err != nil {
		return err
	}
	if err := r.Append(r.lifecycle("recording_stopped", endTimeUnixMS)); err != nil {
		return err
	}
	if r.engine != nil {
		result, err := r.engine.Finish(endTimeUnixMS)
		if err != nil {
			r.state = "failed"
			return err
		}
		if err := r.persistIncidents(result, endTimeUnixMS); err != nil {
			return err
		}
	}
	complete = r.session.Clone()
	complete.State, complete.EndTimeUnixMS = "complete", &endTimeUnixMS
	if err := r.sink.Finalize(complete); err != nil {
		r.state = "failed"
		return err
	}
	r.session, r.state = complete, "complete"
	return r.buffer.Advance(endTimeUnixMS)
}

func (r *Recorder) ConfigureIncidents(config incidents.Config) error {
	if r.state != "idle" {
		return fmt.Errorf("configure incidents before starting a session")
	}
	if err := config.Validate(); err != nil {
		return err
	}
	if config.PreWindowMS > r.buffer.windowMS {
		return fmt.Errorf("rolling buffer is shorter than incident pre-window")
	}
	config.CriticalModules = append([]string{}, config.CriticalModules...)
	r.incidentConfig = &config
	return nil
}

func (r *Recorder) persistIncidents(result incidents.Result, timestamp int64) error {
	writer, ok := r.sink.(interface {
		AppendIncident(incidents.Incident) error
	})
	if !ok {
		r.state = "failed"
		return fmt.Errorf("configured recorder requires incident storage")
	}
	for _, item := range result.Updates {
		if err := writer.AppendIncident(item); err != nil {
			r.state = "failed"
			return err
		}
	}
	for _, notice := range result.Triggers {
		data, err := json.Marshal(map[string]any{"incident_id": notice.IncidentID, "trigger_id": notice.Trigger.TriggerID, "trigger_type": notice.Trigger.Type, "evidence": notice.Trigger.Evidence})
		if err != nil {
			r.state = "failed"
			return err
		}
		event := r.lifecycle("incident_triggered", timestamp)
		event.Event.Attributes = data
		event.Event.Severity = notice.Trigger.Severity
		if _, _, err := r.appendRaw(event); err != nil {
			return err
		}
	}
	for _, id := range result.Finalized {
		data, _ := json.Marshal(map[string]string{"incident_id": id})
		event := r.lifecycle("incident_capture_finalized", timestamp)
		event.Event.Attributes = data
		if _, _, err := r.appendRaw(event); err != nil {
			return err
		}
	}
	r.session.IncidentCount = int64(r.engine.Count())
	return nil
}

func (r *Recorder) MarkIncident(label string, timestamp int64) (string, error) {
	if r.engine == nil {
		return "", fmt.Errorf("incident capture is not configured")
	}
	if strings.TrimSpace(label) == "" || len(label) > 256 {
		return "", fmt.Errorf("manual label must contain 1–256 bytes")
	}
	data, _ := json.Marshal(map[string]string{"label": label})
	event := r.lifecycle("manual_incident_marked", timestamp)
	event.Event.Attributes = data
	expectedSequence := r.session.ObservationCount + r.session.EventCount + 1
	if err := r.Append(event); err != nil {
		return "", err
	}
	for _, item := range r.engine.Snapshot() {
		for _, trigger := range item.Triggers {
			if trigger.Evidence.RecordSequence == expectedSequence {
				return item.IncidentID, nil
			}
		}
	}
	return "", fmt.Errorf("manual mark produced no incident")
}

func (r *Recorder) Incidents() []incidents.Incident {
	if r.engine == nil {
		return []incidents.Incident{}
	}
	return r.engine.Snapshot()
}

func (r *Recorder) Close() error {
	if r.sink == nil {
		return nil
	}
	if r.state == "recording" {
		r.state = "closed"
	}
	return r.sink.Close()
}

func (r *Recorder) State() string                { return r.state }
func (r *Recorder) Session() telemetry.Session   { return r.session.Clone() }
func (r *Recorder) Snapshot() []telemetry.Record { return r.buffer.Snapshot() }

func (r *Recorder) lifecycle(kind string, timestamp int64) telemetry.Record {
	return telemetry.Record{Event: &telemetry.Event{SchemaVersion: 1, TimestampUnixMS: timestamp, SessionID: r.session.SessionID, Source: r.session.Source, EventType: kind, Severity: "info"}}
}
