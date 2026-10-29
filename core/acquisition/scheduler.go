// Package acquisition prepares live acquisition using deterministic offline I/O.
// No mobile package imports it. It has no transport, sockets or wall-clock loop.
package acquisition

import (
	"fmt"
	"math"
	"sort"
	"vehicleflightrecorder.local/core/obd"
)

const Second int64 = 1_000_000_000
const MaxRawBytes = 512 << 10
const MaxTransactions = 512

type Schedule struct {
	Key        string `json:"key"`
	IntervalNS int64  `json:"interval_ns"`
	TimeoutNS  int64  `json:"timeout_ns"`
	Priority   int    `json:"priority"`
}
type Raw struct {
	Sequence    int    `json:"sequence"`
	Generation  int    `json:"generation"`
	Transaction string `json:"transaction_id"`
	Kind        string `json:"kind"`
	NS          int64  `json:"monotonic_ns"`
	Bytes       []byte `json:"bytes_base64"`
}
type Transaction struct {
	ID            string   `json:"id"`
	Generation    int      `json:"generation"`
	Key           string   `json:"key"`
	StartNS       int64    `json:"start_ns"`
	SentNS        *int64   `json:"sent_ns"`
	EndNS         *int64   `json:"end_ns"`
	Outcome       string   `json:"outcome"`
	AcceptedBytes int      `json:"accepted_bytes"`
	RawRefs       []int    `json:"raw_refs"`
	AdapterStatus []string `json:"adapter_status"`
}
type Observation struct {
	Transaction   string  `json:"transaction_id"`
	Generation    int     `json:"generation"`
	ECU           string  `json:"ecu"`
	Signal        string  `json:"signal_id"`
	Value         float64 `json:"value"`
	Unit          string  `json:"unit"`
	ReceiveNS     int64   `json:"receive_ns"`
	MappedMS      int64   `json:"mapped_unix_ms"`
	MeasurementMS *int64  `json:"measurement_unix_ms"`
	Decoder       string  `json:"decoder_version"`
}
type Statistics struct {
	Key          string  `json:"key"`
	ECU          string  `json:"ecu"`
	Count        int     `json:"count"`
	Rate         float64 `json:"rate_hz"`
	MaxGapNS     int64   `json:"maximum_gap_ns"`
	LatencyMinNS int64   `json:"latency_min_ns"`
	LatencyP95NS int64   `json:"latency_p95_ns"`
}
type Report struct {
	Version      int              `json:"report_version"`
	Kind         string           `json:"kind"`
	Source       string           `json:"source"`
	Profile      string           `json:"protocol_profile"`
	Catalog      string           `json:"catalog_version"`
	AnchorMS     int64            `json:"anchor_unix_ms"`
	DurationNS   int64            `json:"duration_ns"`
	Generation   int              `json:"generation"`
	State        string           `json:"state"`
	Vehicle      string           `json:"vehicle_identity"`
	Reference    string           `json:"reference_comparison"`
	Schedule     []Schedule       `json:"schedule"`
	Raw          []Raw            `json:"raw"`
	Transactions []Transaction    `json:"transactions"`
	Observations []Observation    `json:"observations"`
	Capabilities []obd.Capability `json:"capabilities"`
	Statistics   []Statistics     `json:"statistics"`
	Issues       []string         `json:"issues"`
}
type entry struct {
	spec     Schedule
	due      int64
	failures int
}
type Work struct {
	ID         string
	Generation int
	Key        string
	Bytes      []byte
}
type Scheduler struct {
	entries                                   []entry
	report                                    Report
	capabilities                              *obd.Capabilities
	framer                                    obd.Framer
	active, drain                             int
	deadline, drainDeadline, lastNS, nextSend int64
	rawBytes                                  int
}

func New(anchorMS int64, specs []Schedule) (*Scheduler, error) {
	if anchorMS <= 0 || anchorMS > 253402300799999 || len(specs) == 0 || len(specs) > 16 {
		return nil, fmt.Errorf("invalid anchor or schedule size")
	}
	s := &Scheduler{active: -1, drain: -1, report: Report{Version: 1, Kind: "offline-acquisition", Source: "synthetic", Profile: obd.Profile, Catalog: obd.CatalogVersion, AnchorMS: anchorMS, Generation: 1, State: "ready", Vehicle: "unknown", Reference: "unmeasured", Schedule: append([]Schedule(nil), specs...)}}
	seen := map[string]bool{}
	for _, spec := range specs {
		if _, err := obd.Lookup(spec.Key); err != nil {
			return nil, err
		}
		if seen[spec.Key] || spec.IntervalNS < Second/10 || spec.IntervalNS > 60*Second || spec.TimeoutNS < Second/10 || spec.TimeoutNS > 5*Second || spec.Priority < 0 || spec.Priority > 9 {
			return nil, fmt.Errorf("invalid schedule")
		}
		seen[spec.Key] = true
		s.entries = append(s.entries, entry{spec: spec})
	}
	s.capabilities = obd.NewCapabilities("offline-generation-1")
	return s, nil
}
func (s *Scheduler) clock(now int64) error {
	if now < s.lastNS || now < 0 || now > 24*60*60*Second {
		return fmt.Errorf("offline clock regression or 24-hour limit")
	}
	s.lastNS = now
	s.report.DurationNS = now
	return nil
}
func (s *Scheduler) raw(kind, id string, now int64, data []byte) error {
	if len(data) > 65536 || s.rawBytes+len(data) > MaxRawBytes || len(s.report.Raw) >= 4096 {
		s.report.State = "blocked"
		s.report.Issues = append(s.report.Issues, "raw_budget_exhausted")
		return fmt.Errorf("raw evidence budget exhausted")
	}
	s.rawBytes += len(data)
	seq := len(s.report.Raw) + 1
	s.report.Raw = append(s.report.Raw, Raw{seq, s.report.Generation, id, kind, now, append([]byte(nil), data...)})
	for i := range s.report.Transactions {
		if s.report.Transactions[i].ID == id {
			s.report.Transactions[i].RawRefs = append(s.report.Transactions[i].RawRefs, seq)
			break
		}
	}
	return nil
}
func (s *Scheduler) eligible(key string) bool {
	if key == "support" || key == "adapter_voltage" {
		return true
	}
	for _, row := range s.capabilities.Rows() {
		if row.Key == key && (row.Observed || (row.Advertised != nil && *row.Advertised)) {
			return true
		}
	}
	return false
}

// Poll selects one due request; oldest due wins, then priority, then catalog key.
// Each entry has one deadline, not a backlog of missed polling intervals.
func (s *Scheduler) Poll(now int64) (*Work, error) {
	if err := s.Tick(now); err != nil {
		return nil, err
	}
	if s.report.State != "ready" || now < s.nextSend {
		return nil, nil
	}
	selected := -1
	for i, e := range s.entries {
		if e.due > now || !s.eligible(e.spec.Key) {
			continue
		}
		if selected < 0 || e.due < s.entries[selected].due || (e.due == s.entries[selected].due && (e.spec.Priority < s.entries[selected].spec.Priority || (e.spec.Priority == s.entries[selected].spec.Priority && e.spec.Key < s.entries[selected].spec.Key))) {
			selected = i
		}
	}
	if selected < 0 {
		return nil, nil
	}
	if len(s.report.Transactions) >= MaxTransactions {
		s.report.State = "blocked"
		s.report.Issues = append(s.report.Issues, "transaction_budget_exhausted")
		return nil, fmt.Errorf("transaction budget exhausted")
	}
	d, _ := obd.Lookup(s.entries[selected].spec.Key)
	tx := Transaction{ID: fmt.Sprintf("tx-%04d", len(s.report.Transactions)+1), Generation: s.report.Generation, Key: d.Key, StartNS: now, Outcome: "pending"}
	s.report.Transactions = append(s.report.Transactions, tx)
	s.active = len(s.report.Transactions) - 1
	s.report.State = "writing"
	s.deadline = now + s.entries[selected].spec.TimeoutNS
	s.framer.Reset()
	if err := s.raw("requested", tx.ID, now, []byte(d.Command)); err != nil {
		s.finish(now, "resource_limit")
		s.report.State = "blocked"
		return nil, err
	}
	return &Work{tx.ID, tx.Generation, tx.Key, []byte(d.Command)}, nil
}

// Accepted represents actual stream acceptance of the next Go-owned prefix.
// Zero progress is a no-op, not a successful write or a busy retry loop.
func (s *Scheduler) Accepted(generation int, id string, now int64, n int) error {
	if generation != s.report.Generation {
		return fmt.Errorf("stale connection generation")
	}
	if err := s.clock(now); err != nil {
		return err
	}
	if s.active < 0 || s.report.State != "writing" || s.report.Transactions[s.active].ID != id {
		return fmt.Errorf("no matching write")
	}
	tx := &s.report.Transactions[s.active]
	d, _ := obd.Lookup(tx.Key)
	if n < 0 || n > len(d.Command)-tx.AcceptedBytes {
		return fmt.Errorf("invalid accepted count")
	}
	if n > 0 {
		if err := s.raw("accepted", id, now, []byte(d.Command[tx.AcceptedBytes:tx.AcceptedBytes+n])); err != nil {
			s.finish(now, "resource_limit")
			s.report.State = "blocked"
			return err
		}
		tx.AcceptedBytes += n
	}
	if now >= s.deadline {
		s.finish(now, "timeout")
		s.report.State = "blocked"
		return fmt.Errorf("write completed after deadline; delivery uncertain")
	}
	if tx.AcceptedBytes == len(d.Command) {
		value := now
		tx.SentNS = &value
		s.report.State = "awaiting"
		s.deadline = now + s.spec(tx.Key).TimeoutNS
	}
	return nil
}
func (s *Scheduler) spec(key string) Schedule {
	for _, e := range s.entries {
		if e.spec.Key == key {
			return e.spec
		}
	}
	panic("internal missing request")
}
func (s *Scheduler) finish(now int64, outcome string) {
	if s.active < 0 {
		return
	}
	tx := &s.report.Transactions[s.active]
	value := now
	tx.EndNS = &value
	tx.Outcome = outcome
	for i := range s.entries {
		if s.entries[i].spec.Key != tx.Key {
			continue
		}
		e := &s.entries[i]
		if outcome == "ok" || outcome == "partial" {
			e.failures = 0
		} else {
			if e.failures < 5 {
				e.failures++
			}
			for _, row := range s.capabilities.Rows() {
				if row.Key == tx.Key {
					s.capabilities.Unavailable(row.ECU, row.Key, tx.ID)
				}
			}
		}
		delay := e.spec.IntervalNS * (int64(1) << uint(e.failures))
		cap := 30 * Second
		if e.spec.IntervalNS > cap {
			cap = e.spec.IntervalNS
		}
		if delay > cap {
			delay = cap
		}
		e.due = now + delay
	}
	s.active = -1
	s.nextSend = now + Second/10
	s.report.State = "ready"
}
func (s *Scheduler) Tick(now int64) error {
	if err := s.clock(now); err != nil {
		return err
	}
	if s.report.State == "draining" && now >= s.drainDeadline {
		s.report.State = "blocked"
		s.report.Issues = append(s.report.Issues, "drain_deadline_requires_reconnect")
	}
	if s.active >= 0 && now >= s.deadline {
		idx := s.active
		tx := s.report.Transactions[idx]
		s.finish(now, "timeout")
		if tx.SentNS != nil {
			s.drain = idx
			s.report.State = "draining"
			s.drainDeadline = now + Second
		} else if tx.AcceptedBytes > 0 {
			s.report.State = "blocked"
			s.report.Issues = append(s.report.Issues, "partial_command_requires_reconnect")
		}
	}
	return nil
}
func (s *Scheduler) Receive(generation int, now int64, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty received chunk")
	}
	if generation != s.report.Generation {
		return fmt.Errorf("stale connection generation")
	}
	if err := s.Tick(now); err != nil {
		return err
	}
	id := ""
	kind := "unsolicited"
	if s.active >= 0 {
		id = s.report.Transactions[s.active].ID
		kind = "received"
	} else if s.report.State == "draining" {
		id = s.report.Transactions[s.drain].ID
		kind = "late_received"
	}
	if err := s.raw(kind, id, now, data); err != nil {
		s.finish(now, "resource_limit")
		s.report.State = "blocked"
		return err
	}
	if s.report.State == "blocked" || s.report.State == "disconnected" {
		return fmt.Errorf("connection cannot admit responses")
	}
	if s.report.State == "ready" {
		s.report.State = "blocked"
		s.report.Issues = append(s.report.Issues, "unsolicited_bytes_require_reconnect")
		return fmt.Errorf("unsolicited bytes")
	}
	frames, err := s.framer.Feed(data)
	if err != nil {
		s.finish(now, "protocol_error")
		s.report.State = "blocked"
		return err
	}
	if len(frames) == 0 {
		return nil
	}
	if len(frames) != 1 || s.framer.Pending() != 0 {
		s.finish(now, "protocol_error")
		s.report.State = "blocked"
		return fmt.Errorf("unexpected additional response boundary")
	}
	if s.report.State == "draining" {
		s.framer.Reset()
		s.drain = -1
		s.report.State = "ready"
		return nil
	}
	if s.report.State != "awaiting" {
		s.finish(now, "protocol_error")
		s.report.State = "blocked"
		return fmt.Errorf("response before completed write")
	}
	tx := &s.report.Transactions[s.active]
	reply, err := obd.Parse(tx.Key, frames[0])
	if err != nil {
		s.finish(now, "protocol_error")
		return err
	}
	tx.AdapterStatus = append([]string{}, reply.Statuses...)
	for _, failure := range reply.Failures {
		tx.AdapterStatus = append(tx.AdapterStatus, "decode_error:"+failure.ECU+":"+failure.Detail)
		if failure.ECU != "unknown" {
			s.capabilities.Unavailable(failure.ECU, tx.Key, tx.ID)
		}
	}
	hasFailedResponder := false
	outcome := "adapter_error"
	if len(reply.Statuses) == 1 && reply.Statuses[0] == "NO DATA" {
		outcome = "no_data"
	}
	for _, response := range reply.Responses {
		if response.State == "advertised" {
			_ = s.capabilities.ApplySupport(response, tx.ID)
			outcome = "ok"
		}
		if response.State == "negative_response" {
			hasFailedResponder = true
			s.capabilities.Unavailable(response.ECU, tx.Key, tx.ID)
			if outcome != "ok" {
				outcome = "negative_response"
			}
		}
		if response.State == "unavailable" {
			hasFailedResponder = true
			s.capabilities.Unavailable(response.ECU, tx.Key, tx.ID)
			if outcome != "ok" {
				outcome = "unavailable"
			}
		}
		if response.Value != nil {
			if len(s.report.Observations) >= 2048 {
				s.finish(now, "resource_limit")
				s.report.State = "blocked"
				return fmt.Errorf("observation budget exhausted")
			}
			d, _ := obd.Lookup(tx.Key)
			s.report.Observations = append(s.report.Observations, Observation{tx.ID, tx.Generation, response.ECU, d.Signal, *response.Value, d.Unit, now, s.report.AnchorMS + now/1_000_000, nil, obd.DecoderVersion})
			s.capabilities.Observe(response.ECU, tx.Key, tx.ID)
			outcome = "ok"
		}
	}
	if outcome == "ok" && (len(reply.Failures) > 0 || len(reply.Statuses) > 0 || hasFailedResponder) {
		outcome = "partial"
	}
	s.finish(now, outcome)
	return nil
}

// Reconnect is explicit. It cancels old work, discards framing and capability
// cache, and never replays a partly sent command. Clock anchors remain unchanged.
func (s *Scheduler) Reconnect(now int64) error {
	if err := s.clock(now); err != nil {
		return err
	}
	s.finish(now, "cancelled")
	s.report.Generation++
	s.report.State = "ready"
	s.drain = -1
	s.framer.Reset()
	s.nextSend = now
	s.capabilities = obd.NewCapabilities(fmt.Sprintf("offline-generation-%d", s.report.Generation))
	for i := range s.entries {
		s.entries[i].due = now
		s.entries[i].failures = 0
	}
	return nil
}
func (s *Scheduler) Stop(now int64) error {
	if err := s.clock(now); err != nil {
		return err
	}
	s.finish(now, "cancelled")
	s.report.State = "disconnected"
	s.framer.Reset()
	return nil
}

// Report owns its result; callers can inspect/export without mutating the owner.
func (s *Scheduler) Report() Report {
	r := s.report
	r.Capabilities = s.capabilities.Rows()
	r.Statistics = statistics(r)
	if r.Raw == nil {
		r.Raw = []Raw{}
	}
	if r.Transactions == nil {
		r.Transactions = []Transaction{}
	}
	if r.Observations == nil {
		r.Observations = []Observation{}
	}
	if r.Capabilities == nil {
		r.Capabilities = []obd.Capability{}
	}
	if r.Statistics == nil {
		r.Statistics = []Statistics{}
	}
	if r.Issues == nil {
		r.Issues = []string{}
	}
	for i := range r.Transactions {
		if r.Transactions[i].AdapterStatus == nil {
			r.Transactions[i].AdapterStatus = []string{}
		}
	}
	return clone(r)
}
func statistics(r Report) []Statistics {
	type group struct {
		key, ecu         string
		times, latencies []int64
	}
	groups := map[string]*group{}
	txs := map[string]Transaction{}
	for _, tx := range r.Transactions {
		txs[tx.ID] = tx
	}
	for _, o := range r.Observations {
		tx := txs[o.Transaction]
		id := tx.Key + "/" + o.ECU
		g := groups[id]
		if g == nil {
			g = &group{key: tx.Key, ecu: o.ECU}
			groups[id] = g
		}
		g.times = append(g.times, o.ReceiveNS)
		if tx.SentNS != nil {
			g.latencies = append(g.latencies, o.ReceiveNS-*tx.SentNS)
		}
	}
	var ids []string
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var result []Statistics
	for _, id := range ids {
		g := groups[id]
		gap := g.times[0]
		for i := 1; i < len(g.times); i++ {
			if delta := g.times[i] - g.times[i-1]; delta > gap {
				gap = delta
			}
		}
		if tail := r.DurationNS - g.times[len(g.times)-1]; tail > gap {
			gap = tail
		}
		sort.Slice(g.latencies, func(i, j int) bool { return g.latencies[i] < g.latencies[j] })
		rate := 0.0
		if r.DurationNS > 0 {
			rate = float64(len(g.times)) * float64(Second) / float64(r.DurationNS)
		}
		min, p95 := int64(0), int64(0)
		if len(g.latencies) > 0 {
			min = g.latencies[0]
			p95 = g.latencies[int(math.Ceil(.95*float64(len(g.latencies))))-1]
		}
		result = append(result, Statistics{g.key, g.ecu, len(g.times), rate, gap, min, p95})
	}
	return result
}
