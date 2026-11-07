package obd

import (
	"strings"
	"testing"
)

func TestCAN29NormalFixedParsing(t *testing.T) {
	for _, frame := range []string{
		"010C\r18DAF110 04 41 0C 1A F8 00 00 00\r18DAF118 04 41 0C 0F A0\r>",
		"18 DA F1 10 04 41 0C 1A F8\r18 DA F1 18 04 41 0C 0F A0\r>",
	} {
		r, err := ParseProfile(CAN29Profile, "rpm", []byte(frame))
		if err != nil || len(r.Responses) != 2 || r.Responses[0].ECU != "18DAF110" || *r.Responses[0].Value != 1726 || r.Responses[1].ECU != "18DAF118" || *r.Responses[1].Value != 1000 {
			t.Fatalf("%+v %v", r, err)
		}
		if _, err := Parse("rpm", []byte(frame)); err == nil {
			t.Fatal("legacy profile accepted CAN29")
		}
	}
	r, err := ParseProfile(CAN29Profile, "coolant", []byte("18DAF110 03 41 05 7B\r>"))
	if err != nil || *r.Responses[0].Value != 83 {
		t.Fatalf("%+v %v", r, err)
	}
	r, err = ParseProfile(CAN29Profile, "rpm", []byte("18DAF110 03 7F 01 12\r>"))
	if err != nil || r.Responses[0].State != "negative_response" || r.Responses[0].Value != nil {
		t.Fatalf("%+v %v", r, err)
	}
	for _, bad := range []string{"7E8 04 41 0C 1A F8>", "18DB33F1 04 41 0C 1A F8>", "18DA10F1 04 41 0C 1A F8>", "18DAF1F1 04 41 0C 1A F8>", "38DAF110 04 41 0C 1A F8>", "18DAF110 41 0C 1A F8>", "18DAF110 10 13 41 0C 1A F8>", "18DAF110 04 41 0C 1A>", "18DAF110 04 41 05 1A F8>", "18DAF110 04 41 0C GG FF>"} {
		if _, err := ParseProfile(CAN29Profile, "rpm", []byte(bad)); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}
func TestCAN29ResponderIsolationAndCapabilityIDs(t *testing.T) {
	for _, bad := range []string{"18DAF110 04 41 0C 00 00\r18DAF110 04 41 0C 00 00", "18DAF110 04 41 0C 00 00\r18DAF110 broken"} {
		r, err := ParseProfile(CAN29Profile, "rpm", []byte(bad+"\r18DAF118 04 41 0C 0F A0\r>"))
		if err != nil || len(r.Responses) != 1 || r.Responses[0].ECU != "18DAF118" || len(r.Failures) == 0 {
			t.Fatalf("%+v %v", r, err)
		}
	}
	r, err := ParseProfile(CAN29Profile, "support", []byte("18DAF118 06 41 00 00 10 00 00\r18DAF110 06 41 00 08 10 00 00\r>"))
	if err != nil {
		t.Fatal(err)
	}
	c := NewCapabilities("can29-generation-1")
	for _, response := range r.Responses {
		if err = c.ApplySupport(response, "tx-1"); err != nil {
			t.Fatal(err)
		}
	}
	rows := c.Rows()
	if len(rows) != 4 || rows[0].ECU != "18DAF110" || rows[2].ECU != "18DAF118" || c.Get("18DAF118", "coolant").State != "unsupported" || c.Get("18DAF110", "coolant").State != "advertised" {
		t.Fatalf("%+v", rows)
	}
}
func TestCAN29InitializationScope(t *testing.T) {
	steps, err := InitializationFor(CAN29InitializationVersion)
	if err != nil || len(steps) != 21 {
		t.Fatalf("%v %d", err, len(steps))
	}
	joined := ""
	for _, step := range steps {
		joined += step.Command
		if !IsInitialization(step.Key) {
			t.Fatal(step.Key)
		}
		d, err := LookupLive(step.Key)
		if err != nil || d.Command != step.Command {
			t.Fatalf("lookup mismatch %+v %v", d, err)
		}
	}
	if !strings.Contains(joined, "STP34\rATSH18DB33F1\rSTCAF0\rATCRA\rSTFA\rSTPBRR\r") || strings.Contains(joined, "STP33") || strings.Contains(joined, "ATSH7DF") || strings.Contains(joined, "ATSP") {
		t.Fatal(joined)
	}
	for _, bad := range []string{"33\r>", "A7 [AT]\r>", "AUTO\r>", "?\r>"} {
		if _, err = ParseInitialization("init29_protocol_check", []byte(bad)); err == nil {
			t.Fatalf("accepted unverified protocol %q", bad)
		}
	}
	if _, err = ParseInitialization("init29_protocol_check", []byte("STPR\r34\r>")); err != nil {
		t.Fatal(err)
	}
}
func FuzzCAN29Parser(f *testing.F) {
	f.Add([]byte("18DAF110 04 41 0C 1A F8\r>"))
	f.Add([]byte("18DAF118 broken\r>"))
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = ParseProfile(CAN29Profile, "rpm", data) })
}
