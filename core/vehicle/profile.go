package vehicle

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type Manifest struct {
	SchemaVersion    int    `json:"schema_version"`
	PackVersion      string `json:"pack_version"`
	ProfileID        string `json:"profile_id"`
	Manufacturer     string `json:"manufacturer"`
	Model            string `json:"model"`
	ModelYear        int    `json:"model_year"`
	Engine           string `json:"engine"`
	Transmission     string `json:"transmission"`
	NetworkScope     string `json:"network_scope"`
	Status           string `json:"status"`
	ValidatedSignals bool   `json:"validated_signals"`
}

type Signal struct {
	SignalID     string    `json:"signal_id"`
	Unit         string    `json:"unit"`
	Availability string    `json:"availability"`
	IntervalMS   int64     `json:"interval_ms"`
	MockRange    []float64 `json:"mock_range"`
	Provenance   string    `json:"provenance"`
}

type Profile struct {
	Manifest Manifest
	Signals  []Signal
}

func Load(directory string) (Profile, error) {
	var profile Profile
	if err := readJSON(filepath.Join(directory, "manifest.json"), &profile.Manifest); err != nil {
		return profile, err
	}
	var catalog struct {
		SchemaVersion int      `json:"schema_version"`
		PackVersion   string   `json:"pack_version"`
		ProfileID     string   `json:"profile_id"`
		Signals       []Signal `json:"signals"`
	}
	if err := readJSON(filepath.Join(directory, "signals.json"), &catalog); err != nil {
		return profile, err
	}
	m := profile.Manifest
	if m.SchemaVersion != 1 || catalog.SchemaVersion != 1 {
		return profile, fmt.Errorf("unsupported vehicle pack schema_version")
	}
	if catalog.ProfileID != m.ProfileID || catalog.PackVersion != m.PackVersion {
		return profile, fmt.Errorf("catalog identity/version does not match manifest")
	}
	profile.Signals = catalog.Signals
	return profile, profile.Validate()
}

func (profile Profile) Validate() error {
	m := profile.Manifest
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported vehicle pack schema_version")
	}
	if strings.TrimSpace(m.ProfileID) == "" || strings.TrimSpace(m.PackVersion) == "" || strings.TrimSpace(m.Manufacturer) == "" || strings.TrimSpace(m.Model) == "" || m.ModelYear <= 0 {
		return fmt.Errorf("missing required manifest metadata")
	}
	if m.Status != "mock_only" || m.ValidatedSignals {
		return fmt.Errorf("healthy mock generator requires an explicitly mock_only, unvalidated pack")
	}
	if len(profile.Signals) == 0 {
		return fmt.Errorf("mock signal catalog is empty")
	}
	seen := make(map[string]bool)
	for _, signal := range profile.Signals {
		if strings.TrimSpace(signal.SignalID) == "" || seen[signal.SignalID] || strings.TrimSpace(signal.Unit) == "" || strings.TrimSpace(signal.Provenance) == "" || signal.Availability != "mock_only" || signal.IntervalMS <= 0 || signal.IntervalMS%100 != 0 {
			return fmt.Errorf("invalid or duplicate mock signal %q", signal.SignalID)
		}
		if len(signal.MockRange) != 2 || math.IsNaN(signal.MockRange[0]) || math.IsNaN(signal.MockRange[1]) || math.IsInf(signal.MockRange[0], 0) || math.IsInf(signal.MockRange[1], 0) || signal.MockRange[0] > signal.MockRange[1] {
			return fmt.Errorf("invalid mock_range for %s", signal.SignalID)
		}
		seen[signal.SignalID] = true
	}
	return nil
}

func readJSON(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("%s: expected exactly one JSON document", path)
	}
	return nil
}
