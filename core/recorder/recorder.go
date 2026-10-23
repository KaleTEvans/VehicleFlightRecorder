package recorder

import (
	"errors"
	"fmt"
	"time"

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
	buffer  *Buffer
	sink    sink
	session telemetry.Session
	state   string
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
	if r.state != "recording" {
		return fmt.Errorf("append requires recording state, got %s", r.state)
	}
	if err := r.buffer.check(record); err != nil {
		return err
	}
	if record.SessionID() != r.session.SessionID || record.Source() != r.session.Source || record.Timestamp() < r.session.StartTimeUnixMS {
		return fmt.Errorf("record does not belong to active session/source/time range")
	}
	if err := r.sink.Append(record); err != nil {
		r.state = "failed"
		return err
	}
	if err := r.buffer.Add(record); err != nil {
		r.state = "failed"
		return err
	}
	if record.Observation != nil {
		r.session.ObservationCount++
	} else {
		r.session.EventCount++
	}
	return nil
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
	complete.EventCount = r.session.EventCount
	if err := r.sink.Finalize(complete); err != nil {
		r.state = "failed"
		return err
	}
	r.session, r.state = complete, "complete"
	return r.buffer.Advance(endTimeUnixMS)
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
