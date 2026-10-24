package incidents

import (
	"encoding/json"
	"fmt"
	"strings"

	"vehicleflightrecorder.local/core/telemetry"
)

type Detector interface {
	Evaluate(telemetry.Record) ([]Candidate, error)
}

type rules struct {
	config         Config
	voltageLatched bool
	voltageTime    int64
	voltageValue   float64
	counts         map[string]int
	losses         map[string]int64
	multiLatched   bool
	disconnected   bool
	dtcs           map[string]bool
}

func NewDetector(config Config) (Detector, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config.CriticalModules = append([]string{}, config.CriticalModules...)
	return &rules{config: config, counts: map[string]int{}, losses: map[string]int64{}, dtcs: map[string]bool{}}, nil
}

func (d *rules) Evaluate(record telemetry.Record) ([]Candidate, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}
	if o := record.Observation; o != nil {
		if d.disconnected || o.SignalID != d.config.VoltageSignalID || o.Unit != d.config.VoltageUnit || o.Quality != "valid" {
			return nil, nil
		}
		low := o.Value < d.config.LowVoltage
		drop := d.voltageTime > 0 && o.TimestampUnixMS >= d.voltageTime && o.TimestampUnixMS-d.voltageTime <= d.config.DropWindowMS && d.voltageValue-o.Value >= d.config.DropDelta
		result := []Candidate{}
		if !d.voltageLatched && (low || drop) {
			details, _ := json.Marshal(map[string]any{"value": o.Value, "unit": o.Unit, "low_threshold": d.config.LowVoltage, "drop_threshold": d.config.DropDelta, "low": low, "rapid_drop": drop, "synthetic_thresholds": true})
			result = append(result, Candidate{Type: "voltage_excursion", Severity: "warning", Category: "electrical", Details: details})
			d.voltageLatched = true
		} else if d.voltageLatched && !low && !drop && o.Value >= d.config.ResetVoltage {
			d.voltageLatched = false
		}
		d.voltageTime, d.voltageValue = o.TimestampUnixMS, o.Value
		return result, nil
	}
	e := record.Event
	var attrs struct {
		ExpectedResponse bool   `json:"expected_response"`
		Code             string `json:"code"`
		Label            string `json:"label"`
		Category         string `json:"category"`
	}
	switch e.EventType {
	case "diagnostic_timeout", "dtc_observed", "dtc_removed", "manual_incident_marked":
		if err := json.Unmarshal(e.Attributes, &attrs); err != nil {
			return nil, fmt.Errorf("%s attributes: %w", e.EventType, err)
		}
	}
	switch e.EventType {
	case "transport_disconnected":
		if d.disconnected {
			return nil, nil
		}
		d.disconnected = true
		clear(d.counts)
		clear(d.losses)
		d.multiLatched = false
		d.voltageTime = 0
		d.voltageLatched = false
		if d.config.TransportLoss {
			return []Candidate{{Type: "transport_disconnect", Severity: "warning", Category: "communication"}}, nil
		}
	case "transport_connected":
		d.disconnected = false
		d.voltageTime = 0
	case "response_restored":
		delete(d.counts, e.Module)
		delete(d.losses, e.Module)
		if len(d.losses) < 2 {
			d.multiLatched = false
		}
	case "diagnostic_timeout":
		if d.disconnected || !attrs.ExpectedResponse {
			return nil, nil
		}
		critical := false
		for _, module := range d.config.CriticalModules {
			if module == e.Module {
				critical = true
			}
		}
		if !critical {
			return nil, nil
		}
		d.counts[e.Module]++
		if d.counts[e.Module] != d.config.TimeoutCount {
			return nil, nil
		}
		d.losses[e.Module] = e.TimestampUnixMS
		details, _ := json.Marshal(map[string]any{"consecutive_expected_timeouts": d.counts[e.Module], "synthetic_thresholds": true})
		result := []Candidate{{Type: "module_timeout", Severity: "warning", Category: "communication", Module: e.Module, Details: details}}
		recent := 0
		for _, timestamp := range d.losses {
			if e.TimestampUnixMS-timestamp <= d.config.CorrelationWindowMS {
				recent++
			}
		}
		if recent >= 2 && !d.multiLatched {
			result = append(result, Candidate{Type: "multi_module_loss", Severity: "warning", Category: "communication", Module: e.Module})
			d.multiLatched = true
		}
		return result, nil
	case "dtc_observed":
		if d.disconnected {
			return nil, nil
		}
		if strings.TrimSpace(attrs.Code) == "" {
			return nil, fmt.Errorf("dtc_observed requires code")
		}
		key := e.Module + ":" + attrs.Code
		if d.dtcs[key] {
			return nil, nil
		}
		d.dtcs[key] = true
		category := attrs.Category
		if category == "" {
			category = "diagnostics"
		}
		if category != "powertrain" && category != "diagnostics" {
			return nil, fmt.Errorf("unsupported DTC category %q", category)
		}
		return []Candidate{{Type: "new_dtc", Severity: e.Severity, Category: category, Module: e.Module, Label: attrs.Code}}, nil
	case "dtc_removed":
		delete(d.dtcs, e.Module+":"+attrs.Code)
	case "manual_incident_marked":
		if strings.TrimSpace(attrs.Label) == "" {
			return nil, fmt.Errorf("manual marker requires a label")
		}
		return []Candidate{{Type: "manual_marker", Severity: "info", Category: "manual", Manual: true, Label: attrs.Label}}, nil
	}
	return nil, nil
}
