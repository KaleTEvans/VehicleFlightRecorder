package acquisition

import (
	"testing"
	"vehicleflightrecorder.local/core/obd"
)

func startupFixture(t *testing.T) *Scheduler {
	t.Helper()
	s, e := NewLive(1700000000000, []Schedule{{Key: "support", IntervalNS: 60 * Second, TimeoutNS: 2 * Second, Priority: 0}, {Key: "rpm", IntervalNS: 2 * Second, TimeoutNS: 2 * Second, Priority: 1}}, LiveConfiguration{SessionID: "live-test", Origin: "scripted-fixture", OperatingState: "unknown", Workload: "rpm_only", Protocol: "ISO15765-4-CAN29-500", Initialization: obd.CAN29RuntimeInitializationVersion, PolicyVersion: "rpm-activity-1", StartOrigin: "activity_probe", MaximumDurationNS: 300 * Second, StartupSyncVersion: StartupSyncVersion})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.BeginStartup(0); e != nil {
		t.Fatal(e)
	}
	return s
}
func TestStartupQuietRetainsResidualBytesWithoutAcknowledging(t *testing.T) {
	s := startupFixture(t)
	if w, e := s.Poll(0); e != nil || w != nil {
		t.Fatal("admitted before quiet interval", w, e)
	}
	if e := s.Receive(1, Second/2, []byte("18DAF110 04 41 0C 0F A0\r>")); e != nil {
		t.Fatal(e)
	}
	if w, e := s.Poll(Second/2 + 1); e != nil || w != nil {
		t.Fatal("residual bytes enabled request", w, e)
	}
	w, e := s.Poll(Second)
	if e != nil || w == nil || string(w.Bytes) != "STPC\r" {
		t.Fatal(w, e)
	}
	if s.Report().Raw[0].Kind != "unsolicited" || s.Initialized() {
		t.Fatal("residual reply treated as setup")
	}
}
func TestStoppedCloseRetriesOnlyAfterBoundedDelayAndRequiresRealOK(t *testing.T) {
	s := startupFixture(t)
	for retry := 0; retry < 3; retry++ {
		now := int64(1+retry*2) * Second
		w, e := s.Poll(now)
		if e != nil || w == nil || w.Key != "init_close" {
			t.Fatal(w, e)
		}
		if e = s.Accepted(1, w.ID, now+1, len(w.Bytes)); e != nil {
			t.Fatal(e)
		}
		if e = s.Receive(1, now+2, []byte("STPC\rSTOPPED\r\r>")); e != nil {
			t.Fatal(e)
		}
		if s.Initialized() || s.initIndex != 0 {
			t.Fatal("STOPPED acknowledged initialization")
		}
		if retry < 2 {
			if w, e = s.Poll(now + Second/2); e != nil || w != nil {
				t.Fatal("retry did not wait", w, e)
			}
		} else if s.State() != "blocked" {
			t.Fatal("retry budget not enforced")
		}
	}
	s = startupFixture(t)
	w, _ := s.Poll(Second)
	_ = s.Accepted(1, w.ID, Second+1, len(w.Bytes))
	_ = s.Receive(1, Second+2, []byte("STOPPED\r>"))
	w, _ = s.Poll(3 * Second)
	_ = s.Accepted(1, w.ID, 3*Second+1, len(w.Bytes))
	_ = s.Receive(1, 3*Second+2, []byte("OK\r>"))
	if s.initIndex != 1 {
		t.Fatal("valid acknowledgement did not advance")
	}
}
func TestStartupContinuousGarbageIsBoundedAndLegacyStillStrict(t *testing.T) {
	s := startupFixture(t)
	for ns := int64(0); ns < 3*Second; ns += Second / 4 {
		if e := s.Receive(1, ns, []byte("x")); e != nil {
			t.Fatal(e)
		}
	}
	if e := s.Receive(1, 3*Second, []byte("x")); e == nil || s.State() != "blocked" {
		t.Fatal("unbounded startup drain")
	}
	if _, e := obd.ParseInitialization("init_close", []byte("STOPPED\r>")); e == nil {
		t.Fatal("strict parser weakened")
	}
}

func TestStartupNeverShortensInitialWaitOrTruncatesQuietAtLimit(t *testing.T) {
	s := startupFixture(t)
	if e := s.Receive(1, 1, []byte("residue")); e != nil {
		t.Fatal(e)
	}
	if w, e := s.Poll(Second/2 + 1); e != nil || w != nil {
		t.Fatal("initial wait shortened", w, e)
	}
	s = startupFixture(t)
	if e := s.Receive(1, 3*Second-Second/4, []byte("residue")); e != nil {
		t.Fatal(e)
	}
	if w, e := s.Poll(3 * Second); e == nil || w != nil || s.State() != "blocked" {
		t.Fatal("quiet interval truncated at deadline", w, e)
	}
}

// Replay the mixed reply retained in M7-2. The voltage frame belongs to a
// different command, so a coalesced OK cannot establish synchronized setup.
func TestStartupRejectsLateVoltageCoalescedWithCloseAcknowledgement(t *testing.T) {
	s := startupFixture(t)
	w, err := s.Poll(Second)
	if err != nil || w == nil || w.Key != "init_close" {
		t.Fatal(w, err)
	}
	if err := s.Accepted(1, w.ID, Second+1, len(w.Bytes)); err != nil {
		t.Fatal(err)
	}
	err = s.Receive(1, Second+2, []byte("13.78\r\r>OK\r\r>"))
	if err == nil || s.State() != "blocked" || s.initIndex != 0 {
		t.Fatalf("mixed reply did not force a fresh connection: state=%s step=%d error=%v", s.State(), s.initIndex, err)
	}
	if err := s.Reconnect(2 * Second); err != nil {
		t.Fatal(err)
	}
	if err := s.BeginStartup(2 * Second); err != nil {
		t.Fatal(err)
	}
	w, err = s.Poll(3 * Second)
	if err != nil || w == nil || w.Key != "init_close" {
		t.Fatal(w, err)
	}
	if err := s.Accepted(2, w.ID, 3*Second+1, len(w.Bytes)); err != nil {
		t.Fatal(err)
	}
	if err := s.Receive(2, 3*Second+2, []byte("OK\r\r>")); err != nil || s.initIndex != 1 {
		t.Fatalf("clean acknowledgement did not restore setup: step=%d error=%v", s.initIndex, err)
	}
}
