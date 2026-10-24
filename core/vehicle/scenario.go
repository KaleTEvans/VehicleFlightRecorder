package vehicle

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"

	"vehicleflightrecorder.local/core/incidents"
	"vehicleflightrecorder.local/core/telemetry"
)

type ScenarioEvent struct {
	OffsetMS   int64           `json:"offset_ms"`
	EventType  string          `json:"event_type"`
	Module     string          `json:"module,omitempty"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
}
type Override struct {
	SignalID      string  `json:"signal_id"`
	StartOffsetMS int64   `json:"start_offset_ms"`
	EndOffsetMS   int64   `json:"end_offset_ms"`
	Value         float64 `json:"value"`
}
type ScenarioGap struct {
	StartOffsetMS int64 `json:"start_offset_ms"`
	EndOffsetMS   int64 `json:"end_offset_ms"`
}
type Scenario struct {
	SchemaVersion         int             `json:"schema_version"`
	PackVersion           string          `json:"pack_version"`
	ProfileID             string          `json:"profile_id"`
	ScenarioID            string          `json:"scenario_id"`
	Availability          string          `json:"availability"`
	Provenance            string          `json:"provenance"`
	Events                []ScenarioEvent `json:"events"`
	Overrides             []Override      `json:"overrides"`
	Gaps                  []ScenarioGap   `json:"gaps"`
	ExpectedIncidentCount int             `json:"expected_incident_count"`
	ExpectedTriggerTypes  []string        `json:"expected_trigger_types"`
}

func validScenario(name string) bool { return telemetry.ValidScenario(name) }

func LoadIncidentConfig(directory string, profile Profile) (incidents.Config, error) {
	var document struct {
		SchemaVersion int              `json:"schema_version"`
		PackVersion   string           `json:"pack_version"`
		ProfileID     string           `json:"profile_id"`
		Thresholds    incidents.Config `json:"thresholds"`
	}
	if err := readJSON(filepath.Join(directory, "thresholds.json"), &document); err != nil {
		return incidents.Config{}, err
	}
	if document.SchemaVersion != 1 || document.PackVersion != profile.Manifest.PackVersion || document.ProfileID != profile.Manifest.ProfileID {
		return incidents.Config{}, fmt.Errorf("incident rules pack identity/version mismatch")
	}
	return document.Thresholds, document.Thresholds.Validate()
}

func LoadScenario(directory, name string, profile Profile) (Scenario, error) {
	var scenario Scenario
	if !validScenario(name) || name == "healthy_drive" {
		return scenario, fmt.Errorf("unsupported fault scenario %q", name)
	}
	if err := readJSON(filepath.Join(directory, "scenarios", name+".json"), &scenario); err != nil {
		return scenario, err
	}
	if err := scenario.Validate(profile); err != nil {
		return scenario, err
	}
	if scenario.ScenarioID != name {
		return scenario, fmt.Errorf("scenario file identity mismatch")
	}
	return scenario, nil
}

func (s Scenario) Validate(profile Profile) error {
	if s.SchemaVersion != 1 || s.PackVersion != profile.Manifest.PackVersion || s.ProfileID != profile.Manifest.ProfileID || !validScenario(s.ScenarioID) || s.ScenarioID == "healthy_drive" || s.Availability != "mock_only" || s.Provenance == "" {
		return fmt.Errorf("invalid synthetic scenario identity/provenance")
	}
	last := int64(math.MinInt64)
	for _, event := range s.Events {
		if event.OffsetMS < last || event.OffsetMS%100 != 0 || event.EventType == "" {
			return fmt.Errorf("scenario events must be ordered on 100 ms steps")
		}
		last = event.OffsetMS
		if err := (telemetry.Event{SchemaVersion: 1, TimestampUnixMS: 1, SessionID: "validation", Source: "mock", EventType: event.EventType, Severity: "info", Attributes: event.Attributes}).Validate(); err != nil {
			return err
		}
	}
	for n, override := range s.Overrides {
		if override.StartOffsetMS >= override.EndOffsetMS || override.StartOffsetMS%100 != 0 || override.EndOffsetMS%100 != 0 || math.IsNaN(override.Value) || math.IsInf(override.Value, 0) {
			return fmt.Errorf("invalid scenario override")
		}
		found := false
		for _, signal := range profile.Signals {
			if signal.SignalID == override.SignalID && signal.EnabledFor(s.ScenarioID) && override.Value >= signal.MockRange[0] && override.Value <= signal.MockRange[1] {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("override signal/value not present in scenario catalog")
		}
		for _, prior := range s.Overrides[:n] {
			if prior.SignalID == override.SignalID && override.StartOffsetMS < prior.EndOffsetMS && override.EndOffsetMS > prior.StartOffsetMS {
				return fmt.Errorf("overlapping overrides")
			}
		}
	}
	for n, gap := range s.Gaps {
		if gap.StartOffsetMS >= gap.EndOffsetMS || gap.StartOffsetMS%100 != 0 || gap.EndOffsetMS%100 != 0 || (n > 0 && gap.StartOffsetMS < s.Gaps[n-1].EndOffsetMS) {
			return fmt.Errorf("invalid ordered scenario gap")
		}
		start, end := false, false
		for _, event := range s.Events {
			start = start || (event.OffsetMS == gap.StartOffsetMS && event.EventType == "transport_disconnected")
			end = end || (event.OffsetMS == gap.EndOffsetMS && event.EventType == "transport_connected")
		}
		if !start || !end {
			return fmt.Errorf("scenario gap needs matching disconnect/reconnect events")
		}
	}
	return nil
}
