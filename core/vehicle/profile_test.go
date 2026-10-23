package vehicle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const packPath = "../../vehicle-packs/alfa-giulia-qv-2017"

func TestLoadMockProfile(t *testing.T) {
	p, err := Load(packPath)
	if err != nil {
		t.Fatal(err)
	}
	if p.Manifest.ProfileID != "alfa.giulia.qv.2017.2_9t" || len(p.Signals) != 6 || p.Manifest.ValidatedSignals {
		t.Fatal("unexpected profile", p)
	}
}

func TestRejectInvalidPacks(t *testing.T) {
	for _, change := range []struct {
		name string
		file string
		edit func(map[string]any)
	}{
		{"schema", "manifest.json", func(m map[string]any) { m["schema_version"] = 2 }},
		{"identity", "signals.json", func(m map[string]any) { m["profile_id"] = "wrong" }},
		{"verified", "manifest.json", func(m map[string]any) { m["validated_signals"] = true }},
		{"empty", "signals.json", func(m map[string]any) { m["signals"] = []any{} }},
		{"duplicate", "signals.json", func(m map[string]any) { signals := m["signals"].([]any); m["signals"] = append(signals, signals[0]) }},
		{"real-availability", "signals.json", func(m map[string]any) { m["signals"].([]any)[0].(map[string]any)["availability"] = "standard_obd" }},
		{"missing-range", "signals.json", func(m map[string]any) { delete(m["signals"].([]any)[0].(map[string]any), "mock_range") }},
		{"bad-rate", "signals.json", func(m map[string]any) { m["signals"].([]any)[0].(map[string]any)["interval_ms"] = 0 }},
	} {
		t.Run(change.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"manifest.json", "signals.json"} {
				data, err := os.ReadFile(filepath.Join(packPath, name))
				if err != nil {
					t.Fatal(err)
				}
				if name == change.file {
					var object map[string]any
					if err := json.Unmarshal(data, &object); err != nil {
						t.Fatal(err)
					}
					change.edit(object)
					data, err = json.Marshal(object)
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Load(dir); err == nil {
				t.Fatal("invalid pack accepted")
			}
		})
	}
}
