package activity

import "testing"

func sample(tx string, sec int64, rpm float64, generation int) *Evidence {
	return &Evidence{tx, generation, sec * Second, rpm, 3}
}
func TestActivityPolicyBoundaries(t *testing.T) {
	p := New(false)
	step := func(sec int64, e *Evidence, connected bool) Decision {
		t.Helper()
		d, err := p.Step(sec*Second, e, connected)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	if d := step(0, nil, true); d.Action != "" {
		t.Fatal(d)
	}
	for _, s := range []int64{1, 4, 8} {
		if d := step(s, sample("zero", s, 0, 1), true); d.Action != "" || p.Recording {
			t.Fatal(d)
		}
	}
	if d := step(10, sample("rpm-1", 10, 950, 1), true); d.State != "candidate" {
		t.Fatal(d)
	}
	if d := step(14, sample("rpm-1", 10, 950, 1), true); d.Action != "" {
		t.Fatal("duplicate response started drive", d)
	}
	if d := step(15, sample("rpm-2", 15, 950, 1), true); d.Action != "start" {
		t.Fatal(d)
	}
	if d := step(16, nil, false); d.State != "degraded_gap" {
		t.Fatal(d)
	}
	if d := step(40, sample("rpm-new", 40, 950, 2), true); d.Action != "" || !p.Recording {
		t.Fatal("reconnect duplicated drive", d)
	}
	for s := int64(45); s <= 105; s += 5 {
		d := step(s, sample(string(rune(s)), s, 0, 2), true)
		if s < 105 && d.Action != "" {
			t.Fatal(d)
		}
		if s == 105 && d.Action != "end" {
			t.Fatal(d)
		}
	}
}
func TestStaleZeroUnknownAndManualOverride(t *testing.T) {
	p := New(false)
	p.Recording = true
	_, _ = p.Step(0, sample("z", 0, 0, 1), true)
	d, _ := p.Step(60*Second, sample("z", 0, 0, 1), true)
	if d.Action != "" || d.State != "degraded_gap" {
		t.Fatal("stale zero treated as end", d)
	}
	d, _ = p.Step(240*Second, nil, false)
	if d.Action != "end" {
		t.Fatal(d)
	}
	p.ManualStop()
	for s := int64(241); s <= 250; s++ {
		d, _ = p.Step(s*Second, sample("rpm", s, 950, 2), true)
		if d.Action != "" || !p.Suppressed {
			t.Fatal(d)
		}
	}
	for s := int64(251); s <= 311; s += 5 {
		_, _ = p.Step(s*Second, sample("z", s, 0, 2), true)
	}
	if p.Suppressed {
		t.Fatal("sustained off must rearm")
	}
	if _, err := p.Step(0, nil, true); err == nil {
		t.Fatal("clock regression accepted")
	}
}
func TestFreshnessResetAndConflictingResponders(t *testing.T) {
	p := New(false)
	_, _ = p.Step(0, sample("a", 0, 950, 1), true)
	_, _ = p.Step(15*Second, sample("b", 15, 950, 1), true) // gap must reset candidate even if both samples individually fresh
	// Owner ticks detect intervening missing execution; caller supplies a gap reset.
	p = New(false)
	_, _ = p.Step(0, sample("a", 0, 950, 1), true)
	_, _ = p.Step(11*Second, sample("a", 0, 950, 1), true)
	d, _ := p.Step(15*Second, sample("b", 15, 950, 1), true)
	if d.Action != "" {
		t.Fatal("gap bridged debounce", d)
	}
	d, _ = p.Step(20*Second, sample("c", 20, 950, 2), true)
	if d.Action != "" {
		t.Fatal("new generation bridged debounce", d)
	}
}

func TestSuspendedOwnerCannotBridgeDriveGrace(t *testing.T) {
	p := New(false)
	p.Recording = true
	_, _ = p.Step(0, sample("a", 0, 950, 1), true)
	d, e := p.Step(181*Second, sample("b", 181, 950, 1), true)
	if e != nil || d.Action != "end" || d.Reason != "owner_execution_gap_exceeded_grace" {
		t.Fatal(d, e)
	}
}
