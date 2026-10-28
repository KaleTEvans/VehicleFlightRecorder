package mobile

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"vehicleflightrecorder.local/core/incidents"
	"vehicleflightrecorder.local/core/telemetry"
)

func scanJSON(path, name string, visit func(int64, json.RawMessage) error) error {
	file, err := openEvidence(path, name, maxLogBytes)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReaderSize(file, maxJSONBytes+1)
	line := int64(0)
	bytesRead := int64(0)
	for {
		data, err := reader.ReadSlice('\n')
		bytesRead += int64(len(data))
		if bytesRead > maxLogBytes {
			return fault("resource_limit", "scanJSON", fmt.Errorf("%s exceeded file budget while reading", name))
		}
		if err == bufio.ErrBufferFull || len(data) > maxJSONBytes {
			return fault("resource_limit", "scanJSON", fmt.Errorf("%s:%d exceeds JSONL line limit", name, line+1))
		}
		if len(data) == 0 && err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line++
		if len(data) == 0 || data[len(data)-1] != '\n' || len(bytes.TrimSpace(data)) == 0 || !json.Valid(data) {
			return fault("corrupt_data", "scanJSON", fmt.Errorf("%s:%d: malformed/truncated JSONL", name, line))
		}
		if err := visit(line, data); err != nil {
			var known *bridgeFailure
			if !errors.As(err, &known) {
				err = fault("corrupt_data", "scanJSON", err)
			}
			return fmt.Errorf("%s:%d: %w", name, line, err)
		}
		if err != nil {
			return err
		}
	}
}
func readIncidents(path string) ([]incidents.Incident, error) {
	items := []incidents.Incident{}
	indexes := map[string]int{}
	sizes := map[string]int{}
	retainedBytes := 0
	err := scanJSON(path, "incidents.jsonl", func(_ int64, data json.RawMessage) error {
		var item incidents.Incident
		if err := decodeDocument(data, &item); err != nil {
			return err
		}
		if err := item.Validate(); err != nil {
			return fault("corrupt_data", "readIncidents", err)
		}
		if item.SessionID != filepath.Base(path) {
			return fault("corrupt_data", "readIncidents", fmt.Errorf("foreign incident"))
		}
		retainedBytes += len(data) - sizes[item.IncidentID]
		if retainedBytes > maxResponseBytes {
			return fault("resource_limit", "readIncidents", fmt.Errorf("incident snapshot memory budget exceeded"))
		}
		sizes[item.IncidentID] = len(data)
		if n, ok := indexes[item.IncidentID]; ok {
			if item.Revision != items[n].Revision+1 {
				return fmt.Errorf("invalid incident revision")
			}
			items[n] = item
		} else {
			if len(items) >= 2000 {
				return fault("resource_limit", "readIncidents", fmt.Errorf("incident limit reached"))
			}
			if item.Revision != 1 {
				return fmt.Errorf("invalid initial incident revision")
			}
			indexes[item.IncidentID] = len(items)
			items = append(items, item)
		}
		return nil
	})
	return items, err
}

type evidenceRow struct {
	Reference incidents.Reference `json:"reference"`
	Record    json.RawMessage     `json:"record"`
}

// IncidentJSON returns at most 200 canonical rows per page, ordered by recorded
// sequence. Coverage/triggers always come from Go, including active snapshots.
func (c *Controller) IncidentJSON(sessionID, incidentID string, offset int64) (result string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("IncidentJSON", &err)
	if err := c.checkOpen(); err != nil {
		return "", err
	}
	if len(incidentID) == 0 || len(incidentID) > 256 {
		return "", fault("invalid_input", "IncidentJSON", fmt.Errorf("invalid incident ID"))
	}
	if offset < 0 {
		return "", fault("invalid_input", "IncidentJSON", fmt.Errorf("negative timeline offset"))
	}
	path, err := c.sessionPath(sessionID)
	if err != nil {
		return "", err
	}
	items, err := readIncidents(path)
	if err != nil {
		return "", err
	}
	if c.core != nil && c.core.Session().SessionID == sessionID {
		items = c.core.Incidents()
	}
	var selected *incidents.Incident
	for _, item := range items {
		if item.IncidentID == incidentID {
			copy := item
			selected = &copy
			break
		}
	}
	if selected == nil {
		return "", fault("invalid_input", "IncidentJSON", fmt.Errorf("unknown incident ID"))
	}
	rows := []evidenceRow{}
	retainedBytes := 0
	for _, name := range []string{"observations.jsonl", "events.jsonl"} {
		err := scanJSON(path, name, func(line int64, data json.RawMessage) error {
			var record telemetry.Record
			ref := incidents.Reference{File: name, Line: line}
			if name == "observations.jsonl" {
				var o telemetry.Observation
				if err := decodeDocument(data, &o); err != nil {
					return err
				}
				record.Observation = &o
				ref.SignalID = o.SignalID
			} else {
				var e telemetry.Event
				if err := decodeDocument(data, &e); err != nil {
					return err
				}
				record.Event = &e
				ref.EventType = e.EventType
			}
			if err := record.Validate(); err != nil {
				return err
			}
			if record.SessionID() != sessionID {
				return fmt.Errorf("foreign session evidence")
			}
			if record.Timestamp() >= selected.AvailableStartUnixMS && record.Timestamp() <= selected.AvailableEndUnixMS {
				retainedBytes += len(data)
				if retainedBytes > 2*maxResponseBytes {
					return fault("resource_limit", "IncidentJSON", fmt.Errorf("timeline index memory budget exceeded; evidence preserved"))
				}
				if len(rows) >= maxTimelineRows {
					return fault("resource_limit", "IncidentJSON", fmt.Errorf("capture exceeds timeline row limit; original evidence preserved"))
				}
				ref.TimestampUnixMS = record.Timestamp()
				ref.RecordSequence = record.Sequence()
				rows = append(rows, evidenceRow{Reference: ref, Record: append(json.RawMessage{}, data...)})
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i].Reference, rows[j].Reference
		if a.TimestampUnixMS != b.TimestampUnixMS {
			return a.TimestampUnixMS < b.TimestampUnixMS
		}
		return a.RecordSequence < b.RecordSequence
	})
	triggerOffsets := map[string]int64{}
	for n, row := range rows {
		for _, trigger := range selected.Triggers {
			if row.Reference.RecordSequence == trigger.Evidence.RecordSequence {
				triggerOffsets[trigger.TriggerID] = int64(n)
			}
		}
	}
	total := int64(len(rows))
	if offset > total {
		return "", fault("invalid_input", "IncidentJSON", fmt.Errorf("timeline offset exceeds evidence"))
	}
	end := min(total, offset+200)
	return encode(map[string]any{"incident": selected, "trigger_offsets": triggerOffsets, "timeline": rows[offset:end], "total_records": total, "next_offset": end, "has_more": end < total})
}
