package mobile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"vehicleflightrecorder.local/core/telemetry"
)

type historyHeader struct {
	path    string
	session telemetry.Session
}

// SessionsJSON is retained for bridge-1 clients; new clients should page history.
func (c *Controller) SessionsJSON() (result string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("SessionsJSON", &err)
	if err := c.checkOpen(); err != nil {
		return "", err
	}
	headers, err := c.historyHeaders()
	if err != nil {
		return "", err
	}
	rows, err := c.historyRows(headers)
	if err != nil {
		return "", err
	}
	return encode(rows)
}

// SessionsPageJSON returns a bounded page, never an implicitly truncated history.
func (c *Controller) SessionsPageJSON(offset int64) (result string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("SessionsPageJSON", &err)
	if err := c.checkOpen(); err != nil {
		return "", err
	}
	if offset < 0 {
		return "", fault("invalid_input", "SessionsPageJSON", fmt.Errorf("negative offset"))
	}
	headers, err := c.historyHeaders()
	if err != nil {
		return "", err
	}
	total := int64(len(headers))
	if offset > total {
		return "", fault("invalid_input", "SessionsPageJSON", fmt.Errorf("offset exceeds history"))
	}
	end := min(total, offset+historyPageSize)
	rows, err := c.historyRows(headers[offset:end])
	if err != nil {
		return "", err
	}
	return encode(map[string]any{"sessions": rows, "total_sessions": total, "next_offset": end, "has_more": end < total})
}
func (c *Controller) historyHeaders() ([]historyHeader, error) {
	root, err := os.Open(c.root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entries, err := root.ReadDir(maxHistoryEntries + 1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(entries) > maxHistoryEntries {
		return nil, fault("resource_limit", "history", fmt.Errorf("history exceeds %d entries; evidence is preserved", maxHistoryEntries))
	}
	headers := []historyHeader{}
	metadataBytes := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path, err := c.sessionPath(entry.Name())
		if err != nil {
			return nil, err
		}
		session, err := readSession(path)
		if err != nil {
			return nil, fmt.Errorf("session %s: %w", entry.Name(), err)
		}
		encoded, err := json.Marshal(session)
		if err != nil {
			return nil, err
		}
		metadataBytes += len(encoded)
		if metadataBytes > maxResponseBytes {
			return nil, fault("resource_limit", "history", fmt.Errorf("history metadata exceeds memory budget; evidence preserved"))
		}
		headers = append(headers, historyHeader{path, session})
	}
	sort.Slice(headers, func(i, j int) bool {
		if headers[i].session.StartTimeUnixMS == headers[j].session.StartTimeUnixMS {
			return headers[i].session.SessionID > headers[j].session.SessionID
		}
		return headers[i].session.StartTimeUnixMS > headers[j].session.StartTimeUnixMS
	})
	return headers, nil
}
func (c *Controller) historyRows(headers []historyHeader) ([]map[string]any, error) {
	rows := []map[string]any{}
	responseBytes := 0
	for _, header := range headers {
		session, path := header.session, header.path
		items, err := readIncidents(path)
		if err != nil {
			return nil, err
		}
		if c.core != nil && session.SessionID == c.core.Session().SessionID {
			session = c.core.Session()
			items = c.core.Incidents()
		}
		if session.State == "recording" && (c.core == nil || session.SessionID != c.core.Session().SessionID) {
			counts := map[string]int64{}
			for _, name := range []string{"observations.jsonl", "events.jsonl"} {
				if err := scanJSON(path, name, func(_ int64, data json.RawMessage) error {
					var record telemetry.Record
					if name == "observations.jsonl" {
						var value telemetry.Observation
						if err := decodeDocument(data, &value); err != nil {
							return err
						}
						record.Observation = &value
					} else {
						var value telemetry.Event
						if err := decodeDocument(data, &value); err != nil {
							return err
						}
						record.Event = &value
					}
					if err := record.Validate(); err != nil {
						return fault("corrupt_data", "history", err)
					}
					if record.SessionID() != session.SessionID || record.Source() != session.Source {
						return fault("corrupt_data", "history", fmt.Errorf("foreign record"))
					}
					counts[name]++
					return nil
				}); err != nil {
					return nil, err
				}
			}
			session.ObservationCount = counts["observations.jsonl"]
			session.EventCount = counts["events.jsonl"]
			session.IncidentCount = int64(len(items))
		}
		var size int64
		for _, name := range []string{"session.json", "observations.jsonl", "events.jsonl", "incidents.jsonl", "raw/transport.jsonl"} {
			f, err := openEvidence(path, name, maxLogBytes)
			if err != nil {
				return nil, err
			}
			info, err := f.Stat()
			f.Close()
			if err != nil {
				return nil, err
			}
			size += info.Size()
		}
		row := map[string]any{"session": session, "incidents": items, "size_bytes": size}
		encoded, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		responseBytes += len(encoded)
		if responseBytes > maxResponseBytes {
			return nil, fault("resource_limit", "history", fmt.Errorf("history response exceeds memory budget; original evidence is preserved"))
		}
		rows = append(rows, row)
	}
	return rows, nil
}
