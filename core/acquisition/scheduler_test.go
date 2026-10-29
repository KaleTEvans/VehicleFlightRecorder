package acquisition

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func makeScheduler(t *testing.T) *Scheduler {
	t.Helper()
	s, e := New(1793000000000, []Schedule{{"support", 10 * Second, Second / 2, 0}, {"rpm", Second, Second / 2, 1}, {"coolant", Second, Second / 2, 2}, {"adapter_voltage", Second, Second / 2, 3}})
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func response(t *testing.T, s *Scheduler, w *Work, now int64, text string) {
	t.Helper()
	if e := s.Accepted(w.Generation, w.ID, now, len(w.Bytes)); e != nil {
		t.Fatal(e)
	}
	for i, chunk := range []string{text[:len(text)/2], text[len(text)/2:]} {
		if e := s.Receive(w.Generation, now+int64(i+1), []byte(chunk)); e != nil {
			t.Fatal(e)
		}
	}
}
func discover(t *testing.T, s *Scheduler) {
	t.Helper()
	w, e := s.Poll(0)
	if e != nil || w == nil || w.Key != "support" {
		t.Fatal(w, e)
	}
	response(t, s, w, 1, "7E8 06 41 00 08 10 00 00\r7E9 06 41 00 00 00 00 00\r>")
}
func TestSerializedDiscoveryDecodingAndMetrics(t *testing.T) {
	s := makeScheduler(t)
	discover(t, s)
	now := Second
	for _, key := range []string{"rpm", "coolant", "adapter_voltage"} {
		w, e := s.Poll(now)
		if e != nil || w == nil || w.Key != key {
			t.Fatal(w, e)
		}
		if duplicate, _ := s.Poll(now); duplicate != nil {
			t.Fatal("overlap")
		}
		text := "7E8 04 41 0C 1A F8\r>"
		if key == "coolant" {
			text = "7E8 03 41 05 7B\r>"
		}
		if key == "adapter_voltage" {
			text = "12.34\r>"
		}
		response(t, s, w, now+1, text)
		now += Second
	}
	if e := s.Stop(now); e != nil {
		t.Fatal(e)
	}
	r := s.Report()
	if len(r.Observations) != 3 || len(r.Statistics) != 3 {
		t.Fatal(r)
	}
	for _, o := range r.Observations {
		if o.MeasurementMS != nil || o.MappedMS != r.AnchorMS+o.ReceiveNS/1_000_000 {
			t.Fatal(o)
		}
	}
	for _, stats := range r.Statistics {
		if stats.Count != 1 || stats.Rate != .25 {
			t.Fatal(stats)
		}
	}
	r.Raw[0].Bytes[0] = 'X'
	if bytes.Equal(s.Report().Raw[0].Bytes, r.Raw[0].Bytes) {
		t.Fatal("shared report")
	}
	out := filepath.Join(t.TempDir(), "report.json")
	if e := s.WriteReport(out); e != nil {
		t.Fatal(e)
	}
	before, _ := os.ReadFile(out)
	if e := s.WriteReport(out); e == nil {
		t.Fatal("overwrite")
	}
	after, _ := os.ReadFile(out)
	if !bytes.Equal(before, after) {
		t.Fatal("changed prior evidence")
	}
}
func TestTimeoutDrainAndFairness(t *testing.T) {
	s := makeScheduler(t)
	discover(t, s)
	w, _ := s.Poll(Second)
	if w.Key != "rpm" {
		t.Fatal(w)
	}
	if e := s.Accepted(w.Generation, w.ID, Second, len(w.Bytes)); e != nil {
		t.Fatal(e)
	}
	if e := s.Tick(2 * Second); e != nil {
		t.Fatal(e)
	}
	if s.Report().State != "draining" {
		t.Fatal(s.Report().State)
	}
	if next, _ := s.Poll(2 * Second); next != nil {
		t.Fatal("overlap with late reply")
	}
	if e := s.Receive(w.Generation, 2*Second+1, []byte("7E8 04 41 0C 00 00\r>")); e != nil {
		t.Fatal(e)
	}
	if len(s.Report().Observations) != 0 {
		t.Fatal("late observation")
	}
	next, _ := s.Poll(2*Second + Second/5)
	if next == nil || next.Key != "coolant" {
		t.Fatal("starved unrelated request", next)
	}
	response(t, s, next, 2*Second+Second/5+1, "7E8 03 41 05 7B\r>")
	if s.capabilities.Get("7E8", "rpm").State != "unavailable" {
		t.Fatal(s.capabilities.Rows())
	}
}
func TestPartialWriteDeadlineAndGeneration(t *testing.T) {
	s := makeScheduler(t)
	w, _ := s.Poll(0)
	if e := s.Accepted(w.Generation, w.ID, 1, 1); e != nil {
		t.Fatal(e)
	}
	_ = s.Tick(Second)
	if s.Report().State != "blocked" {
		t.Fatal("partial command reused")
	}
	if e := s.Reconnect(2 * Second); e != nil {
		t.Fatal(e)
	}
	next, _ := s.Poll(2 * Second)
	if next == nil || next.Generation == w.Generation || !bytes.Equal(next.Bytes, []byte("0100\r")) {
		t.Fatal(next)
	}
	if e := s.Receive(w.Generation, 2*Second+1, []byte("NO DATA\r>")); e == nil {
		t.Fatal("stale reply")
	}
	if e := s.Accepted(w.Generation, w.ID, 2*Second+1, 1); e == nil {
		t.Fatal("stale write")
	}
	if s.capabilities.Get("7E8", "rpm").State != "unknown" {
		t.Fatal("stale cache")
	}
	_ = s.Stop(3 * Second)
	if s.Report().Transactions[1].Outcome != "cancelled" {
		t.Fatal(s.Report())
	}
}
func TestBoundsMalformedClocksAndNoData(t *testing.T) {
	s := makeScheduler(t)
	w, _ := s.Poll(0)
	response(t, s, w, 1, "NO DATA\r>")
	if s.capabilities.Get("7E8", "rpm").State != "unknown" {
		t.Fatal("timeout means unsupported")
	}
	if w, _ := s.Poll(Second); w == nil || w.Key != "adapter_voltage" {
		t.Fatal("polled unchecked PID", w)
	}
	if _, e := s.Poll(0); e == nil {
		t.Fatal("clock regression")
	}
	s = makeScheduler(t)
	w, _ = s.Poll(0)
	_ = s.Accepted(w.Generation, w.ID, 1, len(w.Bytes))
	if e := s.Receive(w.Generation, 2, bytes.Repeat([]byte{'X'}, 8193)); e == nil {
		t.Fatal("frame overflow")
	}
	if s.Report().State != "blocked" {
		t.Fatal("overflow not blocked")
	}
	s = makeScheduler(t)
	w, _ = s.Poll(0)
	_ = s.Accepted(w.Generation, w.ID, 1, len(w.Bytes))
	s.rawBytes = MaxRawBytes
	if e := s.Receive(w.Generation, 2, []byte(">")); e == nil {
		t.Fatal("raw budget")
	}
	if s.Report().State != "blocked" {
		t.Fatal("resource fault hidden")
	}
}
func TestDrainDeadlineRequiresReconnect(t *testing.T) {
	s := makeScheduler(t)
	w, _ := s.Poll(0)
	_ = s.Accepted(w.Generation, w.ID, 1, len(w.Bytes))
	_ = s.Tick(Second)
	_ = s.Tick(3 * Second)
	if next, _ := s.Poll(4 * Second); next != nil || s.Report().State != "blocked" {
		t.Fatal("late old reply could contaminate new request")
	}
}

func TestPartialECUFailureKeepsHealthyMeasurements(t *testing.T) {
	s := makeScheduler(t)
	discover(t, s)
	w, _ := s.Poll(Second)
	response(t, s, w, Second+1, "7E8 04 41 0C 1A F8\r7E9 04 41 0C 00\r>")
	r := s.Report()
	if len(r.Observations) != 1 || r.Observations[0].ECU != "7E8" || r.Transactions[1].Outcome != "partial" {
		t.Fatal(r)
	}
	if s.capabilities.Get("7E8", "rpm").State != "observed" {
		t.Fatal("healthy ECU invalidated")
	}
}

func TestFailureNeverSpeedsUpLongConfiguredInterval(t *testing.T) {
	s, e := New(1793000000000, []Schedule{{Key: "adapter_voltage", IntervalNS: 60 * Second, TimeoutNS: Second, Priority: 0}})
	if e != nil {
		t.Fatal(e)
	}
	w, _ := s.Poll(0)
	response(t, s, w, 1, "NO DATA\r>")
	if work, _ := s.Poll(31 * Second); work != nil {
		t.Fatal("failure increased request rate")
	}
	if work, _ := s.Poll(61 * Second); work == nil {
		t.Fatal("request never retried")
	}
}

func TestTransactionBudgetStopsWithoutDiscardingHistory(t *testing.T) {
	s, e := New(1793000000000, []Schedule{{Key: "adapter_voltage", IntervalNS: Second / 10, TimeoutNS: Second, Priority: 0}})
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < MaxTransactions; i++ {
		now := int64(i) * Second
		w, e := s.Poll(now)
		if e != nil || w == nil {
			t.Fatal(i, w, e)
		}
		response(t, s, w, now+1, "12.34\r>")
	}
	if _, e = s.Poll(int64(MaxTransactions) * Second); e == nil {
		t.Fatal("history limit ignored")
	}
	report := s.Report()
	if len(report.Transactions) != MaxTransactions || report.Transactions[0].ID != "tx-0001" || report.State != "blocked" {
		t.Fatal("history discarded or admission not stopped")
	}
}
