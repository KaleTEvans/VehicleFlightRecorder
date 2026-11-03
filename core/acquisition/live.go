package acquisition

import (
	"fmt"
	"vehicleflightrecorder.local/core/obd"
)

// NewLive retains the offline engine's ordered admission/failure invariants but
// uses an explicit live schema, setup gate and separately bounded test budget.
func NewLive(anchorMS int64, specs []Schedule, config LiveConfiguration) (*Scheduler, error) {
	if config.SessionID == "" || config.NativeAnchorNS < 0 || config.Initialization != obd.InitializationVersion || config.Protocol != "ISO15765-4-CAN11-500" {
		return nil, fmt.Errorf("invalid live configuration")
	}
	if config.Origin != "physical" && config.Origin != "native-fixture" && config.Origin != "scripted-fixture" {
		return nil, fmt.Errorf("invalid live origin")
	}
	s, err := New(anchorMS, specs)
	if err != nil {
		return nil, err
	}
	s.live = true
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
	for _, d := range obd.Initialization() {
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
func (s *Scheduler) Initialized() bool               { return s.live && s.initIndex == len(obd.Initialization()) }
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
func (s *Scheduler) CapabilityRows() []obd.Capability { return s.capabilities.Rows() }
