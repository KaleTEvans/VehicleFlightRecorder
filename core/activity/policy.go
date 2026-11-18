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
	EndGraceNS      int64   `json:"end_grace_ns"`
	UnknownGraceNS  int64   `json:"unknown_grace_ns"`
}

func Default() Configuration {
	return Configuration{Version, 400, 10 * Second, 4 * Second, 60 * Second, 180 * Second}
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
	previousReceive                   int64
	Config                            Configuration
	Recording, Suppressed             bool
	state, transaction                string
	candidate, zero, unknown, lastNow int64
	generation, positives             int
}

func New(suppressed bool) *Policy {
	return &Policy{Config: Default(), Suppressed: suppressed, candidate: -1, zero: -1, unknown: -1, lastNow: -1, previousReceive: -1}
}
func (p *Policy) ManualStop() {
	p.Recording = false
	p.Suppressed = true
	p.candidate = -1
	p.zero = -1
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
		p.zero = -1
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
		p.zero = -1
	}
	distinct := fresh && e.Transaction != p.transaction
	if distinct {
		if p.previousReceive >= 0 && e.ReceiveNS-p.previousReceive > p.Config.FreshNS {
			p.candidate = -1
			p.positives = 0
			p.zero = -1
		}
		p.previousReceive = e.ReceiveNS
		p.transaction = e.Transaction
	}
	d := Decision{State: "waiting_for_activity", Reason: "running_unknown"}
	if !fresh {
		p.candidate = -1
		p.positives = 0
		p.zero = -1
		if p.Recording {
			if p.unknown < 0 {
				p.unknown = now
			}
			d.State = "degraded_gap"
			d.Reason = "activity_missing_or_stale"
			if now-p.unknown >= p.Config.UnknownGraceNS {
				p.Recording = false
				d = Decision{"finalized", "end", "activity_unknown_grace_expired"}
			}
		}
	} else if e.MaximumRPM >= p.Config.RunningRPM {
		p.unknown = -1
		p.zero = -1
		if p.Recording {
			d = Decision{"recording", "", "validated_running"}
		} else if !p.Suppressed {
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
	} else if e.MaximumRPM == 0 {
		p.unknown = -1
		p.candidate = -1
		p.positives = 0
		if p.zero < 0 {
			p.zero = e.ReceiveNS
		}
		d = Decision{"waiting_for_activity", "", "validated_engine_off"}
		if p.Recording {
			d.State = "end_grace"
		}
		// A fresh response must actually span the grace: a timer cannot turn an old
		// last zero into a sustained validated engine-off condition.
		if e.ReceiveNS-p.zero >= p.Config.EndGraceNS {
			if p.Recording {
				p.Recording = false
				d = Decision{"finalized", "end", "sustained_zero_rpm"}
			}
			p.Suppressed = false
		}
	} else {
		p.candidate = -1
		p.positives = 0
		p.zero = -1
		if p.Recording {
			if p.unknown < 0 {
				p.unknown = now
			}
			d = Decision{"degraded_gap", "", "rpm_below_running_threshold"}
			if now-p.unknown >= p.Config.UnknownGraceNS {
				p.Recording = false
				d = Decision{"finalized", "end", "activity_unknown_grace_expired"}
			}
		}
	}
	if p.Suppressed && !p.Recording {
		d.State = "manual_stop_override"
		d.Reason = "waiting_for_sustained_engine_off_or_rearm"
	}
	p.state = d.State
	return d, nil
}
