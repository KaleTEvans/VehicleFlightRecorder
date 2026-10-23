package mock

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"

	"vehicleflightrecorder.local/core/telemetry"
	"vehicleflightrecorder.local/core/vehicle"
)

func profile(t *testing.T) vehicle.Profile {
	t.Helper()
	p, err := vehicle.Load("../../vehicle-packs/alfa-giulia-qv-2017")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestHealthyDeterministicRatesAndPhases(t *testing.T) {
	p := profile(t)
	c := Config{SessionID: "deterministic", StartTimeUnixMS: 1000, Seed: 42, DurationMS: 120000}
	generate := func(seed int64) [32]byte {
		config := c
		config.Seed = seed
		hash := sha256.New()
		counts := make(map[string]int)
		phases := []string{}
		previous := int64(0)
		err := Healthy(context.Background(), config, p, func(record telemetry.Record) error {
			if err := record.Validate(); err != nil {
				return err
			}
			if record.Timestamp() < previous {
				t.Fatal("generator reordered logical time")
			}
			previous = record.Timestamp()
			if record.Observation != nil {
				counts[record.Observation.SignalID]++
				if record.Observation.Source != "mock" || record.Observation.VehiclePackVersion != p.Manifest.PackVersion {
					t.Fatal("mock provenance missing")
				}
			} else {
				var attrs map[string]string
				if err := json.Unmarshal(record.Event.Attributes, &attrs); err != nil {
					return err
				}
				phases = append(phases, attrs["phase"])
			}
			return json.NewEncoder(hash).Encode(record)
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, signal := range p.Signals {
			if counts[signal.SignalID] != int(c.DurationMS/signal.IntervalMS+1) {
				t.Fatalf("wrong sample count for %s: %d", signal.SignalID, counts[signal.SignalID])
			}
		}
		want := []string{"engine_start", "idle", "acceleration", "cruise", "deceleration", "stop", "shutdown"}
		if len(phases) != len(want) {
			t.Fatal("wrong phases", phases)
		}
		for i := range want {
			if phases[i] != want[i] {
				t.Fatal("wrong phase ordering", phases)
			}
		}
		return [32]byte(hash.Sum(nil))
	}
	a := generate(42)
	if a != generate(42) {
		t.Fatal("fixed inputs changed generator output")
	}
	if a == generate(43) {
		t.Fatal("different seeds produced identical waveforms")
	}
}

func TestHealthyFailureAndCancellation(t *testing.T) {
	p := profile(t)
	c := Config{SessionID: "s", StartTimeUnixMS: 1000, Seed: 42, DurationMS: 120000}
	failure := errors.New("sink failed")
	if !errors.Is(Healthy(context.Background(), c, p, func(telemetry.Record) error { return failure }), failure) {
		t.Fatal("generator swallowed sink failure")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(Healthy(ctx, c, p, func(telemetry.Record) error { t.Fatal("emitted after cancellation"); return nil }), context.Canceled) {
		t.Fatal("cancellation ignored")
	}
	p.Signals[0].Unit = "mph"
	if Healthy(context.Background(), c, p, func(telemetry.Record) error { t.Fatal("emitted with mismatched units"); return nil }) == nil {
		t.Fatal("invalid unit accepted")
	}
	c.DurationMS = 1000
	if _, err := c.Session(p); err == nil {
		t.Fatal("too-short scenario accepted")
	}
}
