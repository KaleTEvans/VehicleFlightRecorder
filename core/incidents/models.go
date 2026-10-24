package incidents

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Config contains synthetic test rules, not validated vehicle limits.
type Config struct {
	Availability        string   `json:"availability"`
	Provenance          string   `json:"provenance"`
	PreWindowMS         int64    `json:"pre_window_ms"`
	PostWindowMS        int64    `json:"post_window_ms"`
	MaxSilenceMS        int64    `json:"max_silence_ms"`
	VoltageSignalID     string   `json:"voltage_signal_id"`
	VoltageUnit         string   `json:"voltage_unit"`
	LowVoltage          float64  `json:"low_voltage"`
	ResetVoltage        float64  `json:"reset_voltage"`
	DropDelta           float64  `json:"drop_delta"`
	DropWindowMS        int64    `json:"drop_window_ms"`
	TimeoutCount        int      `json:"timeout_count"`
	CriticalModules     []string `json:"critical_modules"`
	CorrelationWindowMS int64    `json:"correlation_window_ms"`
	TransportLoss       bool     `json:"transport_loss"`
}

func (c Config) Validate() error {
	if c.Availability != "mock_only" || strings.TrimSpace(c.Provenance) == "" {
		return fmt.Errorf("incident rules require explicit mock_only provenance")
	}
	if c.PreWindowMS < 0 || c.PostWindowMS < 0 || c.PreWindowMS > 86400000 || c.PostWindowMS > 86400000 || c.MaxSilenceMS <= 0 || c.MaxSilenceMS > 86400000 {
		return fmt.Errorf("invalid incident capture/silence windows")
	}
	if c.VoltageSignalID == "" || c.VoltageUnit == "" || c.LowVoltage <= 0 || c.ResetVoltage <= c.LowVoltage || c.DropDelta <= 0 || c.DropWindowMS <= 0 || c.TimeoutCount < 1 || c.CorrelationWindowMS < 0 {
		return fmt.Errorf("invalid incident detector thresholds")
	}
	for _, value := range []float64{c.LowVoltage, c.ResetVoltage, c.DropDelta} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("thresholds must be finite")
		}
	}
	seen := map[string]bool{}
	for _, module := range c.CriticalModules {
		if strings.TrimSpace(module) == "" || seen[module] {
			return fmt.Errorf("invalid critical module %q", module)
		}
		seen[module] = true
	}
	if len(seen) == 0 {
		return fmt.Errorf("critical modules are required")
	}
	return nil
}

type Reference struct {
	File            string `json:"file"`
	Line            int64  `json:"line"`
	RecordSequence  int64  `json:"record_sequence"`
	TimestampUnixMS int64  `json:"timestamp_unix_ms"`
	SignalID        string `json:"signal_id,omitempty"`
	EventType       string `json:"event_type,omitempty"`
}

func (r Reference) Validate() error {
	if (r.File != "observations.jsonl" && r.File != "events.jsonl") || r.Line < 1 || r.RecordSequence < 1 || r.TimestampUnixMS < 1 {
		return fmt.Errorf("invalid source evidence reference")
	}
	if r.File == "observations.jsonl" && (r.SignalID == "" || r.EventType != "") {
		return fmt.Errorf("observation reference requires signal_id only")
	}
	if r.File == "events.jsonl" && (r.EventType == "" || r.SignalID != "") {
		return fmt.Errorf("event reference requires event_type only")
	}
	return nil
}

type Candidate struct {
	Type     string
	Severity string
	Category string
	Module   string
	Manual   bool
	Label    string
	Details  json.RawMessage
}

type Trigger struct {
	TriggerID       string          `json:"trigger_id"`
	Type            string          `json:"trigger_type"`
	TimestampUnixMS int64           `json:"timestamp_unix_ms"`
	Severity        string          `json:"severity"`
	Category        string          `json:"category"`
	Module          string          `json:"module,omitempty"`
	Manual          bool            `json:"manual"`
	Label           string          `json:"label,omitempty"`
	Evidence        Reference       `json:"evidence"`
	Details         json.RawMessage `json:"details,omitempty"`
}

type Gap struct {
	StartTimeUnixMS int64  `json:"start_time_unix_ms"`
	EndTimeUnixMS   int64  `json:"end_time_unix_ms"`
	Reason          string `json:"reason"`
}

type Incident struct {
	SchemaVersion        int       `json:"schema_version"`
	IncidentID           string    `json:"incident_id"`
	SessionID            string    `json:"session_id"`
	Revision             int       `json:"revision"`
	TriggerTimeUnixMS    int64     `json:"trigger_time_unix_ms"`
	TriggerType          string    `json:"trigger_type"`
	PreWindowSeconds     float64   `json:"pre_window_seconds"`
	PostWindowSeconds    float64   `json:"post_window_seconds"`
	WindowStartUnixMS    int64     `json:"window_start_unix_ms"`
	WindowEndUnixMS      int64     `json:"window_end_unix_ms"`
	AvailableStartUnixMS int64     `json:"available_start_unix_ms"`
	AvailableEndUnixMS   int64     `json:"available_end_unix_ms"`
	Severity             string    `json:"severity"`
	Status               string    `json:"status"`
	Manual               bool      `json:"manual"`
	Triggers             []Trigger `json:"triggers"`
	RelatedIncidentIDs   []string  `json:"related_incident_ids"`
	CoverageGaps         []Gap     `json:"coverage_gaps"`
	IncompleteReasons    []string  `json:"incomplete_reasons"`
}

func (i Incident) Validate() error {
	if i.SchemaVersion != 1 || i.IncidentID == "" || i.SessionID == "" || i.Revision < 1 || len(i.Triggers) == 0 {
		return fmt.Errorf("invalid incident identity/version/triggers")
	}
	if i.PreWindowSeconds < 0 || i.PostWindowSeconds < 0 || i.PreWindowSeconds > 86400 || i.PostWindowSeconds > 86400 || math.Mod(i.PreWindowSeconds*1000, 1) != 0 || math.Mod(i.PostWindowSeconds*1000, 1) != 0 || math.IsNaN(i.PreWindowSeconds) || math.IsNaN(i.PostWindowSeconds) || math.IsInf(i.PreWindowSeconds, 0) || math.IsInf(i.PostWindowSeconds, 0) {
		return fmt.Errorf("invalid incident windows")
	}
	if i.AvailableStartUnixMS < 1 || i.AvailableEndUnixMS < i.AvailableStartUnixMS || i.AvailableStartUnixMS < i.WindowStartUnixMS || i.AvailableEndUnixMS > i.WindowEndUnixMS {
		return fmt.Errorf("invalid available coverage")
	}
	if i.Status != "capturing" && i.Status != "captured" && i.Status != "partial" {
		return fmt.Errorf("invalid capture status")
	}
	first := i.Triggers[0]
	if i.TriggerTimeUnixMS != first.TimestampUnixMS || i.TriggerType != first.Type || i.WindowStartUnixMS != first.TimestampUnixMS-int64(i.PreWindowSeconds*1000) || i.AvailableStartUnixMS > first.TimestampUnixMS {
		return fmt.Errorf("incident primary trigger/window mismatch")
	}
	seen := map[string]bool{}
	rank, manual, latest, sequence := 0, false, int64(0), int64(0)
	for _, trigger := range i.Triggers {
		if trigger.TriggerID == "" || seen[trigger.TriggerID] || trigger.Type == "" || trigger.Category == "" || severityRank(trigger.Severity) == 0 || trigger.TimestampUnixMS != trigger.Evidence.TimestampUnixMS || trigger.TimestampUnixMS < latest || trigger.Evidence.RecordSequence < sequence {
			return fmt.Errorf("invalid incident trigger/order")
		}
		if err := trigger.Evidence.Validate(); err != nil {
			return err
		}
		if len(trigger.Details) > 0 {
			var object map[string]json.RawMessage
			if json.Unmarshal(trigger.Details, &object) != nil || object == nil {
				return fmt.Errorf("trigger details must be an object")
			}
		}
		seen[trigger.TriggerID] = true
		rank = max(rank, severityRank(trigger.Severity))
		manual = manual || trigger.Manual
		latest = trigger.TimestampUnixMS
		sequence = trigger.Evidence.RecordSequence
	}
	post := int64(i.PostWindowSeconds * 1000)
	if post < 0 || latest > math.MaxInt64-post || i.WindowEndUnixMS != latest+post || latest > i.AvailableEndUnixMS || rank != severityRank(i.Severity) || manual != i.Manual {
		return fmt.Errorf("incident aggregate/window mismatch")
	}
	previous := i.AvailableStartUnixMS
	for _, gap := range i.CoverageGaps {
		if gap.StartTimeUnixMS < previous || gap.EndTimeUnixMS <= gap.StartTimeUnixMS || gap.EndTimeUnixMS > i.AvailableEndUnixMS || (gap.Reason != "record_silence" && gap.Reason != "transport_disconnected") {
			return fmt.Errorf("invalid coverage gap")
		}
		previous = gap.EndTimeUnixMS
	}
	reasons := []string{}
	if i.AvailableStartUnixMS > i.WindowStartUnixMS {
		reasons = append(reasons, "pre_window_unavailable")
	}
	if i.AvailableEndUnixMS < i.WindowEndUnixMS {
		reasons = append(reasons, "post_window_unavailable")
	}
	if len(i.CoverageGaps) > 0 {
		reasons = append(reasons, "coverage_gap")
	}
	if len(reasons) != len(i.IncompleteReasons) {
		return fmt.Errorf("coverage reasons mismatch")
	}
	for n := range reasons {
		if reasons[n] != i.IncompleteReasons[n] {
			return fmt.Errorf("coverage reasons mismatch")
		}
	}
	if (i.Status == "captured" && len(reasons) > 0) || (i.Status == "partial" && len(reasons) == 0) {
		return fmt.Errorf("capture status mismatch")
	}
	links := map[string]bool{}
	for _, id := range i.RelatedIncidentIDs {
		if id == "" || id == i.IncidentID || links[id] {
			return fmt.Errorf("invalid incident link")
		}
		links[id] = true
	}

	return nil
}

func (i Incident) Clone() Incident {
	i.Triggers = append([]Trigger{}, i.Triggers...)
	for n := range i.Triggers {
		i.Triggers[n].Details = append(json.RawMessage(nil), i.Triggers[n].Details...)
	}
	i.RelatedIncidentIDs = append([]string{}, i.RelatedIncidentIDs...)
	i.CoverageGaps = append([]Gap{}, i.CoverageGaps...)
	i.IncompleteReasons = append([]string{}, i.IncompleteReasons...)
	return i
}

func severityRank(value string) int {
	switch value {
	case "info":
		return 1
	case "warning":
		return 2
	case "error":
		return 3
	}
	return 0
}
