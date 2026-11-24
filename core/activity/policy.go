// Package activity defines the versioned, vehicle-test configuration used by M7.
// It consumes validated aggregate RPM responses, never adapter presence or UI time.
package activity

import "fmt"

const Second int64 = 1_000_000_000
const Version = "rpm-activity-1"

type Configuration struct {
	Version         string  `json:"version"`
	RunningRPM      float64 `json:"running_rpm"`
	FreshNS         int64   `json:"fresh_ns"`
	StartDebounceNS int64   `json:"start_debounce_ns"`
	ZeroReplies     int     `json:"zero_replies"`
	UnknownGraceNS  int64   `json:"unknown_grace_ns"`
}

func Default() Configuration {
	return Configuration{Version, 400, 10 * Second, 4 * Second, 3, 60 * Second}
}

type Evidence struct {
	Transaction string  `json:"transaction_id"`
	Generation  int     `json:"generation"`
	ReceiveNS   int64   `json:"receive_ns"`
	MaximumRPM  float64 `json:"maximum_rpm"`
	Responders  int     `json:"responders"`
}
type Decision struct{ State, Action, Reason string }
type Policy struct {
	previousReceive              int64
	Config                       Configuration
	Recording, Suppressed        bool
	state, transaction           string
	candidate, unknown, lastNow  int64
	generation, positives, zeros int
	gapReason                    string
}

func New(suppressed bool) *Policy {
	return &Policy{Config: Default(), Suppressed: suppressed, candidate: -1, unknown: -1, lastNow: -1, previousReceive: -1}
}
func (p *Policy) ManualStop() {
	p.Recording = false
	p.Suppressed = true
	p.candidate = -1
	p.zeros = 0
	p.unknown = -1
	p.positives = 0
}
func (p *Policy) Step(now int64, e *Evidence, connected bool) (Decision, error) {
	if now < 0 || now < p.lastNow {
		return Decision{}, fmt.Errorf("activity clock regression")
	}
	previousTick := p.lastNow
	p.lastNow = now
	if previousTick >= 0 && now-previousTick > p.Config.FreshNS {
		p.candidate = -1
		p.positives = 0
		p.zeros = 0
	}
	if p.Recording && p.unknown >= 0 && now-p.unknown >= p.Config.UnknownGraceNS {
		p.Recording = false
		return Decision{"finalized", "end", p.gapReason}, nil
	}
	if p.Recording && previousTick >= 0 && now-previousTick > p.Config.UnknownGraceNS {
		p.Recording = false
		return Decision{"finalized", "end", "owner_execution_gap_exceeded_grace"}, nil
	}
	fresh := connected && e != nil && e.Generation > 0 && e.Responders > 0 && e.Transaction != "" && e.ReceiveNS >= 0 && e.ReceiveNS <= now && now-e.ReceiveNS <= p.Config.FreshNS && e.MaximumRPM >= 0 && e.MaximumRPM <= 16383.75
	if fresh && e.Generation != p.generation {
		p.generation = e.Generation
		p.transaction = ""
		p.candidate = -1
		p.positives = 0
		p.zeros = 0
	}
	distinct := fresh && e.Transaction != p.transaction
	if distinct {
		if p.previousReceive >= 0 && e.ReceiveNS-p.previousReceive > p.Config.FreshNS {
			p.candidate = -1
			p.positives = 0
			p.zeros = 0
		}
		p.previousReceive = e.ReceiveNS
		p.transaction = e.Transaction
	}
	d := Decision{State: "waiting_for_activity", Reason: "running_unknown"}
	if !fresh {
		p.candidate = -1
		p.positives = 0
		p.zeros = 0
		if p.Recording {
			if p.unknown < 0 {
				p.unknown = now
				p.gapReason = "rpm_unavailable"
				if !connected {
					p.gapReason = "scanner_disconnected"
				}
			}
			d = Decision{"degraded_gap", "", p.gapReason}
		}
	} else if e.MaximumRPM > 0 {
		p.unknown = -1
		p.zeros = 0
		if p.Recording {
			d = Decision{"recording", "", "validated_running"}
		} else if !p.Suppressed && e.MaximumRPM >= p.Config.RunningRPM {
			if distinct {
				if p.candidate < 0 {
					p.candidate = e.ReceiveNS
					p.positives = 0
				}
				p.positives++
			}
			d = Decision{"candidate", "", "running_debounce"}
			if p.positives >= 2 && e.ReceiveNS-p.candidate >= p.Config.StartDebounceNS {
				p.Recording = true
				d = Decision{"starting", "start", "fresh_repeated_rpm"}
			}
		}
		if e.MaximumRPM < p.Config.RunningRPM {
			p.candidate = -1
			p.positives = 0
		}
	} else {
		p.unknown = -1
		p.candidate = -1
		p.positives = 0
		if distinct {
			p.zeros++
		}
		d = Decision{"waiting_for_activity", "", "validated_engine_off"}
		if p.Recording {
			d.State = "end_grace"
		}
		if p.zeros >= p.Config.ZeroReplies {
			if p.Recording {
				p.Recording = false
				d = Decision{"finalized", "end", "three_zero_rpm_replies"}
			}
			p.Suppressed = false
		}
	}
	if p.Suppressed && !p.Recording {
		d.State = "manual_stop_override"
		d.Reason = "waiting_for_sustained_engine_off_or_rearm"
	}
	p.state = d.State
	return d, nil
}
