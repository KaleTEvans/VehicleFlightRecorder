package mobile

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"vehicleflightrecorder.local/core/incidents"
	"vehicleflightrecorder.local/core/telemetry"
)

func scanJSON(path, name string, visit func(int64, json.RawMessage) error) error {
	file, err := os.Open(filepath.Join(path, name))
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	line := int64(0)
	for {
		data, err := reader.ReadBytes('\n')
		if len(data) == 0 && err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line++
		if len(data) == 0 || data[len(data)-1] != '\n' || len(bytes.TrimSpace(data)) == 0 || !json.Valid(data) {
			return fmt.Errorf("%s:%d: malformed/truncated JSONL", name, line)
		}
		if err := visit(line, data); err != nil {
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
	err := scanJSON(path, "incidents.jsonl", func(_ int64, data json.RawMessage) error {
		var item incidents.Incident
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		if err := item.Validate(); err != nil {
			return err
		}
		if n, ok := indexes[item.IncidentID]; ok {
			if item.Revision != items[n].Revision+1 {
				return fmt.Errorf("invalid incident revision")
			}
			items[n] = item
		} else {
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
func (c *Controller) IncidentJSON(sessionID, incidentID string, offset int64) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if offset < 0 {
		return "", fmt.Errorf("negative timeline offset")
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
		return "", fmt.Errorf("unknown incident ID")
	}
	rows := []evidenceRow{}
	for _, name := range []string{"observations.jsonl", "events.jsonl"} {
		err := scanJSON(path, name, func(line int64, data json.RawMessage) error {
			var record telemetry.Record
			ref := incidents.Reference{File: name, Line: line}
			if name == "observations.jsonl" {
				var o telemetry.Observation
				if err := json.Unmarshal(data, &o); err != nil {
					return err
				}
				record.Observation = &o
				ref.SignalID = o.SignalID
			} else {
				var e telemetry.Event
				if err := json.Unmarshal(data, &e); err != nil {
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
	total := int64(len(rows))
	if offset > total {
		return "", fmt.Errorf("timeline offset exceeds evidence")
	}
	end := min(total, offset+200)
	return encode(map[string]any{"incident": selected, "timeline": rows[offset:end], "total_records": total, "next_offset": end, "has_more": end < total})
}
