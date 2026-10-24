package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"vehicleflightrecorder.local/core/incidents"
	"vehicleflightrecorder.local/core/telemetry"
)

// WriteJSONL emits one record in one checked write, without hidden buffering.
func WriteJSONL(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return err
}

type Writer struct {
	incidents         *os.File
	incidentRevisions map[string]incidents.Incident
	sequenced         bool
	directory         string
	initial           telemetry.Session
	observations      *os.File
	events            *os.File
	files             []*os.File
	closed            bool
	failed            bool
	last              int64
	observationCount  int64
	eventCount        int64
}

// Create refuses any existing directory. Partial initialization is preserved as
// incomplete evidence rather than deleting files after a filesystem failure.
func Create(directory string, session telemetry.Session) (*Writer, error) {
	if err := session.Validate(); err != nil {
		return nil, err
	}
	if session.State != "recording" || session.ObservationCount != 0 || session.EventCount != 0 || session.IncidentCount != 0 {
		return nil, fmt.Errorf("new session must be recording with zero counts")
	}
	if err := os.MkdirAll(filepath.Dir(directory), 0700); err != nil {
		return nil, err
	}
	if err := os.Mkdir(directory, 0700); err != nil {
		return nil, fmt.Errorf("create new session directory: %w", err)
	}
	w := &Writer{directory: directory, initial: session.Clone(), last: session.StartTimeUnixMS, incidentRevisions: map[string]incidents.Incident{}}
	if err := writeMetadata(directory, session); err != nil {
		return nil, err
	}
	for _, name := range []string{"observations.jsonl", "events.jsonl", "incidents.jsonl", "raw/transport.jsonl"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(directory, name)), 0700); err != nil {
			return nil, errors.Join(err, w.Close())
		}
		file, err := os.OpenFile(filepath.Join(directory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, errors.Join(err, w.Close())
		}
		w.files = append(w.files, file)
	}
	w.observations, w.events = w.files[0], w.files[1]
	w.incidents = w.files[2]
	return w, nil
}

func (w *Writer) Append(record telemetry.Record) error {
	if w.closed || w.failed {
		return fmt.Errorf("session writer is closed or failed")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	count := w.observationCount + w.eventCount
	if count == 0 {
		w.sequenced = record.Sequence() > 0
	}
	if (w.sequenced && record.Sequence() != count+1) || (!w.sequenced && record.Sequence() != 0) {
		return fmt.Errorf("record sequence must be contiguous or entirely absent")
	}
	if record.SessionID() != w.initial.SessionID || record.Source() != w.initial.Source || record.Timestamp() < w.last {
		return fmt.Errorf("session/source mismatch or out-of-order timestamp")
	}
	if record.Observation != nil && record.Observation.VehiclePackVersion != "" && record.Observation.VehiclePackVersion != w.initial.VehiclePackVersion {
		return fmt.Errorf("observation vehicle_pack_version mismatch")
	}
	if w.initial.Mock != nil && record.Timestamp() > w.initial.StartTimeUnixMS+w.initial.Mock.DurationMS {
		return fmt.Errorf("record exceeds configured mock duration")
	}
	var err error
	if record.Observation != nil {
		err = WriteJSONL(w.observations, record.Observation)
	} else {
		err = WriteJSONL(w.events, record.Event)
	}
	if err != nil {
		w.failed = true
		return fmt.Errorf("append session record: %w", err)
	}
	w.last = record.Timestamp()
	if record.Observation != nil {
		w.observationCount++
	} else {
		w.eventCount++
	}
	return nil
}

func (w *Writer) Finalize(session telemetry.Session) error {
	if w.closed || w.failed {
		return fmt.Errorf("cannot finalize closed or failed writer")
	}
	if err := session.Validate(); err != nil {
		return err
	}
	if !reflect.DeepEqual(session.IncidentCapture, w.initial.IncidentCapture) || session.IncidentCount != int64(len(w.incidentRevisions)) {
		return fmt.Errorf("incident metadata/count mismatch")
	}
	for _, item := range w.incidentRevisions {
		if item.Status == "capturing" {
			return fmt.Errorf("incident capture must be finalized before session completion")
		}
	}
	if (session.Mock == nil) != (w.initial.Mock == nil) || (session.Mock != nil && *session.Mock != *w.initial.Mock) {
		return fmt.Errorf("final metadata changed mock configuration")
	}
	if session.State != "complete" || session.SessionID != w.initial.SessionID || session.StartTimeUnixMS != w.initial.StartTimeUnixMS || session.ProfileID != w.initial.ProfileID || session.VehiclePackVersion != w.initial.VehiclePackVersion || session.Source != w.initial.Source || *session.EndTimeUnixMS < w.last || session.ObservationCount != w.observationCount || session.EventCount != w.eventCount {
		return fmt.Errorf("final metadata does not match written session")
	}
	var syncErr error
	for _, file := range w.files {
		syncErr = errors.Join(syncErr, file.Sync())
	}
	err := errors.Join(syncErr, w.Close())
	if err != nil {
		return fmt.Errorf("flush/close session before finalization: %w", err)
	}
	return writeMetadata(w.directory, session)
}

func (w *Writer) AppendIncident(item incidents.Incident) error {
	if w.closed || w.failed {
		return fmt.Errorf("incident writer is closed or failed")
	}
	if err := item.Validate(); err != nil {
		return err
	}
	if item.SessionID != w.initial.SessionID || w.initial.IncidentCapture == nil {
		return fmt.Errorf("incident session/capture mismatch")
	}
	previous := w.incidentRevisions[item.IncidentID]
	if item.Revision != previous.Revision+1 {
		return fmt.Errorf("incident revision must increment by one")
	}
	c := w.initial.IncidentCapture
	if item.PreWindowSeconds*1000 != float64(c.PreWindowMS) || item.PostWindowSeconds*1000 != float64(c.PostWindowMS) || item.AvailableStartUnixMS < w.initial.StartTimeUnixMS || item.AvailableEndUnixMS > w.last {
		return fmt.Errorf("incident coverage/configuration mismatch")
	}
	if previous.Revision > 0 {
		if item.TriggerTimeUnixMS != previous.TriggerTimeUnixMS || item.TriggerType != previous.TriggerType || item.WindowStartUnixMS != previous.WindowStartUnixMS || item.AvailableStartUnixMS != previous.AvailableStartUnixMS || len(item.Triggers) < len(previous.Triggers) || !reflect.DeepEqual(item.Triggers[:len(previous.Triggers)], previous.Triggers) || item.AvailableEndUnixMS < previous.AvailableEndUnixMS {
			return fmt.Errorf("incident revision rewrote evidence")
		}
		if previous.Status != "capturing" {
			copy := item.Clone()
			copy.Revision = previous.Revision
			copy.RelatedIncidentIDs = previous.RelatedIncidentIDs
			if !reflect.DeepEqual(copy, previous) {
				return fmt.Errorf("finalized capture changed")
			}
		}
	}
	for _, trigger := range item.Triggers {
		ref := trigger.Evidence
		limit := w.eventCount
		if ref.File == "observations.jsonl" {
			limit = w.observationCount
		}
		if ref.Line > limit || ref.RecordSequence > w.eventCount+w.observationCount {
			return fmt.Errorf("incident references unwritten evidence")
		}
	}
	if err := WriteJSONL(w.incidents, item); err != nil {
		w.failed = true
		return fmt.Errorf("append incident: %w", err)
	}
	w.incidentRevisions[item.IncidentID] = item.Clone()
	return nil
}

// Close without Finalize deliberately leaves state=recording on disk.
func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	var result error
	for _, file := range w.files {
		result = errors.Join(result, file.Close())
	}
	return result
}

func writeMetadata(directory string, session telemetry.Session) (err error) {
	file, err := os.CreateTemp(directory, ".session-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := os.Remove(file.Name()); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
			err = errors.Join(err, cleanupErr)
		}
	}()
	if encodeErr := WriteJSONL(file, session); encodeErr != nil {
		return errors.Join(encodeErr, file.Close())
	}
	if syncErr := file.Sync(); syncErr != nil {
		return errors.Join(syncErr, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), filepath.Join(directory, "session.json")); err != nil {
		return err
	}
	// Rename consumes the temp file; its absence is expected during cleanup.
	return nil
}
