package acquisition

import (
	"fmt"
	"vehicleflightrecorder.local/core/obd"
)

// NewLive retains the offline engine's ordered admission/failure invariants but
// uses an explicit live schema, setup gate and separately bounded test budget.
func NewLive(anchorMS int64, specs []Schedule, config LiveConfiguration) (*Scheduler, error) {
	if config.SessionID == "" || config.NativeAnchorNS < 0 || config.Protocol != "ISO15765-4-CAN11-500" {
		return nil, fmt.Errorf("invalid live configuration")
	}
	if config.Origin != "physical" && config.Origin != "native-fixture" && config.Origin != "scripted-fixture" {
		return nil, fmt.Errorf("invalid live origin")
	}
	steps, err := obd.InitializationFor(config.Initialization)
	if err != nil {
		return nil, err
	}
	diagnostic := config.Workload == obd.DiagnosticWorkload
	if diagnostic != (config.Initialization == obd.DiagnosticInitializationVersion) {
		return nil, fmt.Errorf("diagnostic workload/version mismatch")
	}
	for _, spec := range specs {
		if obd.IsInitialization(spec.Key) || (spec.Key == "can_status" && !diagnostic) {
			return nil, fmt.Errorf("unrequested diagnostic command")
		}
	}
	s, err := newScheduler(anchorMS, specs, true)
	if err != nil {
		return nil, err
	}
	s.live = true
	s.initSteps = steps
	s.maxTransactions = 2048
	s.maxRaw = 2 << 20
	s.maxRawRows = 16384
	s.maxObservations = 8192
	s.report.Version = 2
	s.report.Kind = "live-acquisition"
	s.report.Source = "obd-live"
	s.report.Catalog = obd.LiveCatalogVersion
	s.report.Configuration = &config
	init := []entry{}
	for _, d := range steps {
		init = append(init, entry{spec: Schedule{d.Key, 60 * Second, 2 * Second, 0}})
	}
	s.entries = append(init, s.entries...)
	s.report.Schedule = nil
	for _, e := range s.entries {
		s.report.Schedule = append(s.report.Schedule, e.spec)
	}
	return s, nil
}
func (s *Scheduler) lookup(key string) (obd.Definition, error) {
	if s.live {
		return obd.LookupLive(key)
	}
	return obd.Lookup(key)
}
func (s *Scheduler) SetRawSink(sink func(Raw) error) { s.rawSink = sink }
func (s *Scheduler) State() string                   { return s.report.State }
func (s *Scheduler) Generation() int                 { return s.report.Generation }
func (s *Scheduler) Initialized() bool               { return s.live && s.initIndex == len(s.initSteps) }
func (s *Scheduler) ObservationCount() int           { return len(s.report.Observations) }
func (s *Scheduler) ObservationsSince(offset int) []Observation {
	if offset < 0 || offset > len(s.report.Observations) {
		return nil
	}
	return append([]Observation(nil), s.report.Observations[offset:]...)
}
func (s *Scheduler) TransactionFor(id string) (Transaction, bool) {
	for _, tx := range s.report.Transactions {
		if tx.ID == id {
			tx.RawRefs = append([]int(nil), tx.RawRefs...)
			tx.AdapterStatus = append([]string(nil), tx.AdapterStatus...)
			return tx, true
		}
	}
	return Transaction{}, false
}
func (s *Scheduler) AddConnection(now int64, metadata map[string]string) error {
	if !s.live {
		return fmt.Errorf("not live")
	}
	if err := s.clock(now); err != nil {
		return err
	}
	copy := map[string]string{}
	for k, v := range metadata {
		copy[k] = v
	}
	s.report.Configuration.Connections = append(s.report.Configuration.Connections, Connection{s.report.Generation, now, copy})
	return nil
}
func (s *Scheduler) CapabilityRows() []obd.Capability {
	rows := s.capabilities.Rows()
	if rows == nil {
		return []obd.Capability{}
	}
	return rows
}

// DiscoverySummary is a UI snapshot of current-generation evidence only. It
// never admits work and never turns NO DATA into unsupported sensor evidence.
func (s *Scheduler) DiscoverySummary(now int64) map[string]any {
	attempts := 0
	outcome := "not_attempted"
	status := ""
	can := ""
	age := int64(-1)
	next := int64(-1)
	for _, tx := range s.report.Transactions {
		if tx.Generation != s.report.Generation {
			continue
		}
		if tx.Key == "support" {
			attempts++
			outcome = tx.Outcome
			status = ""
			if len(tx.AdapterStatus) > 0 {
				status = tx.AdapterStatus[0]
			}
		}
		if tx.Key == "can_status" && tx.EndNS != nil {
			can = tx.Outcome
			age = (now - *tx.EndNS) / 1000000
			if tx.Outcome == "ok" && len(tx.AdapterStatus) == 1 {
				can = tx.AdapterStatus[0]
			}
		}
	}
	requested := false
	for _, e := range s.entries {
		if e.spec.Key == "support" {
			requested = true
		}
	}
	if !requested {
		outcome = "not_requested"
	}
	if !s.Initialized() {
		outcome = "waiting_for_setup"
	}
	if s.Initialized() && outcome != "pending" && s.report.State != "disconnected" && s.report.State != "blocked" {
		for _, e := range s.entries {
			if e.spec.Key == "support" {
				next = (e.due - now) / 1000000
				if next < 0 {
					next = 0
				}
			}
		}
	}
	return map[string]any{"attempts": attempts, "outcome": outcome, "adapter_status": status, "next_attempt_ms": next, "can_status": can, "can_status_age_ms": age, "setup_complete": s.initIndex, "setup_total": len(s.initSteps)}
}
