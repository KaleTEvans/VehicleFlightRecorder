package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"

	"vehicleflightrecorder.local/core/telemetry"
	"vehicleflightrecorder.local/core/vehicle"
)

const StepMS int64 = 100

type Config struct {
	Scenario        string
	SessionID       string
	StartTimeUnixMS int64
	Seed            int64
	DurationMS      int64
}

func (c Config) Session(profile vehicle.Profile) (telemetry.Session, error) {
	if err := profile.Validate(); err != nil {
		return telemetry.Session{}, err
	}
	if c.DurationMS < 120000 || c.DurationMS > 86400000 || c.DurationMS%1000 != 0 || c.StartTimeUnixMS > math.MaxInt64-c.DurationMS {
		return telemetry.Session{}, fmt.Errorf("healthy duration must be whole seconds between 2 minutes and 24 hours, without timestamp overflow")
	}
	session := telemetry.Session{
		SchemaVersion: 1, SessionID: c.SessionID, Source: "mock", ProfileID: profile.Manifest.ProfileID,
		VehiclePackVersion: profile.Manifest.PackVersion, StartTimeUnixMS: c.StartTimeUnixMS, State: "recording",
		Mock: &telemetry.MockConfig{Scenario: "healthy_drive", Seed: c.Seed, DurationMS: c.DurationMS, StepMS: StepMS, GeneratorVersion: "1"},
	}
	name := c.Scenario
	if name == "" {
		name = "healthy_drive"
	}
	session.Mock.Scenario = name
	if name != "healthy_drive" {
		session.Mock.GeneratorVersion = "2"
	}
	return session, session.Validate()
}

// Healthy emits normalized synthetic evidence. No adapter commands or byte stream
// exist. Rates come from the mock-only pack; simulated time never sleeps.
func Healthy(ctx context.Context, config Config, profile vehicle.Profile, emit func(telemetry.Record) error) error {
	if config.Scenario != "" && config.Scenario != "healthy_drive" {
		return fmt.Errorf("use Run for fault scenarios")
	}
	return Run(ctx, config, profile, nil, emit)
}

func Run(ctx context.Context, config Config, profile vehicle.Profile, scenario *vehicle.Scenario, emit func(telemetry.Record) error) error {
	if emit == nil {
		return fmt.Errorf("record emitter is required")
	}
	if _, err := config.Session(profile); err != nil {
		return err
	}
	name := config.Scenario
	if name == "" {
		name = "healthy_drive"
	}
	if name != "healthy_drive" && scenario == nil {
		return fmt.Errorf("fault scenario configuration is required")
	}
	if scenario != nil {
		if err := scenario.Validate(profile); err != nil {
			return err
		}
		if scenario.ScenarioID != name {
			return fmt.Errorf("scenario/config mismatch")
		}
	}
	active := []vehicle.Signal{}
	for _, signal := range profile.Signals {
		if signal.EnabledFor(name) {
			active = append(active, signal)
		}
	}
	if len(profile.Signals) == 0 {
		return fmt.Errorf("healthy scenario requires mock signals")
	}
	for _, signal := range active {
		if _, ok := values(0, 0, 0)[signal.SignalID]; !ok {
			return fmt.Errorf("no healthy waveform for %s", signal.SignalID)
		}
		units := map[string]string{"powertrain.engine.rpm": "rpm", "powertrain.vehicle.speed": "km/h", "powertrain.throttle.position": "%", "electrical.adapter_voltage": "V", "powertrain.engine.calculated_load": "%", "powertrain.coolant.temperature": "degC", "network.tcm_response_latency_ms": "ms", "engine.boost.requested": "kPa", "engine.boost.actual": "kPa"}
		if signal.Unit != units[signal.SignalID] {
			return fmt.Errorf("healthy waveform unit mismatch for %s", signal.SignalID)
		}
		if signal.IntervalMS <= 0 || signal.IntervalMS%StepMS != 0 || len(signal.MockRange) != 2 {
			return fmt.Errorf("invalid signal configuration")
		}
	}
	random := rand.New(rand.NewSource(config.Seed))
	previousPhase := ""
	if scenario != nil {
		for _, event := range scenario.Events {
			if config.DurationMS/2+event.OffsetMS < 0 || config.DurationMS/2+event.OffsetMS > config.DurationMS {
				return fmt.Errorf("scenario event outside drive")
			}
		}
		for _, gap := range scenario.Gaps {
			if config.DurationMS/2+gap.StartOffsetMS < 0 || config.DurationMS/2+gap.EndOffsetMS > config.DurationMS {
				return fmt.Errorf("scenario gap outside drive")
			}
		}
		for _, value := range scenario.Overrides {
			if config.DurationMS/2+value.StartOffsetMS < 0 || config.DurationMS/2+value.EndOffsetMS > config.DurationMS {
				return fmt.Errorf("scenario override outside drive")
			}
		}
	}
	for elapsed := int64(0); elapsed <= config.DurationMS; elapsed += StepMS {
		if err := ctx.Err(); err != nil {
			return err
		}
		timestamp := config.StartTimeUnixMS + elapsed
		offset := elapsed - config.DurationMS/2
		if scenario != nil {
			for _, event := range scenario.Events {
				if event.OffsetMS == offset {
					e := telemetry.Event{SchemaVersion: 1, TimestampUnixMS: timestamp, SessionID: config.SessionID, Source: "mock", EventType: event.EventType, Module: event.Module, Severity: "warning", Attributes: event.Attributes}
					if err := emit(telemetry.Record{Event: &e}); err != nil {
						return err
					}
				}
			}
		}
		progress := float64(elapsed) / float64(config.DurationMS)
		phase := phaseAt(progress)
		if phase != previousPhase {
			attributes, err := json.Marshal(map[string]string{"phase": phase})
			if err != nil {
				return err
			}
			event := telemetry.Event{SchemaVersion: 1, TimestampUnixMS: timestamp, SessionID: config.SessionID, Source: "mock", EventType: "mock_phase_changed", Severity: "info", Attributes: attributes}
			if err := emit(telemetry.Record{Event: &event}); err != nil {
				return err
			}
			previousPhase = phase
		}
		noise := random.Float64()*2 - 1
		current := values(progress, float64(elapsed)/1000, noise)
		gap := false
		if scenario != nil {
			for _, override := range scenario.Overrides {
				if offset >= override.StartOffsetMS && offset < override.EndOffsetMS {
					current[override.SignalID] = override.Value
				}
			}
			for _, missing := range scenario.Gaps {
				if offset >= missing.StartOffsetMS && offset < missing.EndOffsetMS {
					gap = true
				}
			}
		}
		if gap {
			continue
		}
		for _, signal := range active {
			if elapsed%signal.IntervalMS != 0 {
				continue
			}
			value := current[signal.SignalID]
			if value < signal.MockRange[0] || value > signal.MockRange[1] {
				return fmt.Errorf("synthetic %s outside pack mock_range", signal.SignalID)
			}
			observation := telemetry.Observation{SchemaVersion: 1, TimestampUnixMS: timestamp, SessionID: config.SessionID, SignalID: signal.SignalID, Source: "mock", Value: math.Round(value*1000) / 1000, Unit: signal.Unit, Quality: "valid", VehiclePackVersion: profile.Manifest.PackVersion}
			if err := emit(telemetry.Record{Observation: &observation}); err != nil {
				return err
			}
		}
	}
	return nil
}

func phaseAt(progress float64) string {
	switch {
	case progress == 0:
		return "engine_start"
	case progress < 0.15:
		return "idle"
	case progress < 0.30:
		return "acceleration"
	case progress < 0.75:
		return "cruise"
	case progress < 0.90:
		return "deceleration"
	case progress < 1:
		return "stop"
	default:
		return "shutdown"
	}
}

func values(progress, seconds, noise float64) map[string]float64 {
	speed, throttle := 0.0, 5.0
	switch {
	case progress >= 0.15 && progress < 0.30:
		speed, throttle = (progress-0.15)/0.15*85, 45
	case progress >= 0.30 && progress < 0.75:
		speed, throttle = 85+2*math.Sin(seconds/10), 22
	case progress >= 0.75 && progress < 0.90:
		speed, throttle = (0.90-progress)/0.15*85, 2
	}
	rpm, voltage, load := 800+speed*24+noise*8, 13.7+noise*0.03, 15+throttle*0.7
	if progress >= 1 {
		rpm, throttle, voltage, load = 0, 0, 12.6, 0
	}
	return map[string]float64{
		"network.tcm_response_latency_ms":   10,
		"engine.boost.requested":            100 + throttle*2,
		"engine.boost.actual":               100 + throttle*2,
		"powertrain.engine.rpm":             rpm,
		"powertrain.vehicle.speed":          speed,
		"powertrain.throttle.position":      throttle,
		"electrical.adapter_voltage":        voltage,
		"powertrain.engine.calculated_load": load,
		"powertrain.coolant.temperature":    25 + 65*math.Min(progress/0.5, 1),
	}
}
