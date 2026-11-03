package obd

import (
	"bytes"
	"strings"
	"testing"
)

func TestDocumentedExamplesAndZeroes(t *testing.T) {
	for _, test := range []struct {
		key, line, ecu string
		value          float64
	}{
		{"rpm", "010C\r7E8 04 41 0C 1A F8 00 00 00\r>", "7E8", 1726},
		{"coolant", "7E8 03 41 05 7B\r>", "7E8", 83},
		{"rpm", "7E8 04 41 0C 00 00\r>", "7E8", 0},
		{"coolant", "7E8 03 41 05 00\r>", "7E8", -40},
		{"adapter_voltage", "STVR\r12.34\r>", "adapter", 12.34},
		{"adapter_voltage", "0.00\r>", "adapter", 0},
	} {
		r, e := Parse(test.key, []byte(test.line))
		if e != nil || len(r.Responses) != 1 || r.Responses[0].ECU != test.ecu || r.Responses[0].Value == nil || *r.Responses[0].Value != test.value {
			t.Fatalf("%s: %#v %v", test.key, r, e)
		}
	}
}
func TestArbitraryChunkBoundariesAndOwnedFrames(t *testing.T) {
	data := []byte("010C\r7E8 04 41 0C 1A F8\r7E9 04 41 0C 00 00\r>")
	for cut := 0; cut <= len(data); cut++ {
		var f Framer
		first, e := f.Feed(data[:cut])
		if e != nil {
			t.Fatal(e)
		}
		second, e := f.Feed(data[cut:])
		if e != nil {
			t.Fatal(e)
		}
		all := append(first, second...)
		if len(all) != 1 || !bytes.Equal(all[0], data) {
			t.Fatal(cut)
		}
		reply, e := Parse("rpm", all[0])
		if e != nil || len(reply.Responses) != 2 {
			t.Fatal(reply, e)
		}
	}
	var f Framer
	frames, e := f.Feed([]byte("NO DATA\r>STOPPED\r>"))
	if e != nil || len(frames) != 2 {
		t.Fatal(frames, e)
	}
	_, _ = f.Feed([]byte("different>"))
	if string(frames[0]) != "NO DATA\r>" {
		t.Fatal("shared buffer")
	}
}
func TestErrorsAndBounds(t *testing.T) {
	for _, line := range []string{"", ">", "41 0C 1A F8>", "7E8 10 13 41 0C 1A F8>", "18DAF11004410C1AF8>", "7E8 04 41 05 1A F8>", "7E8 04 41 0C 1A>", "7E8 04 41 0C GG FF>", "7E8 04 41 0C 00 00\r7E8 04 41 0C 00 00>", "NO DATA\r>>", strings.Repeat("X", 257) + ">"} {
		if _, e := Parse("rpm", []byte(line)); e == nil {
			t.Fatal("accepted", line)
		}
	}
	for _, line := range []string{"NaN>", "Inf>", "99.99>", "12.345>", "12.34V>"} {
		if _, e := Parse("adapter_voltage", []byte(line)); e == nil {
			t.Fatal(line)
		}
	}
	var f Framer
	if _, e := f.Feed(bytes.Repeat([]byte{'X'}, 8193)); e == nil {
		t.Fatal("unbounded frame")
	}
	if _, e := f.Feed([]byte(">")); e == nil {
		t.Fatal("not poisoned")
	}
	f.Reset()
	if _, e := f.Feed([]byte{0xff}); e == nil {
		t.Fatal("invalid ASCII")
	}
	f.Reset()
	if _, e := f.Feed(bytes.Repeat([]byte{'>'}, 17)); e == nil {
		t.Fatal("unbounded frames")
	}
	for _, key := range []string{"ATZ", "04", "010C\rATZ", "unknown"} {
		if _, e := Lookup(key); e == nil {
			t.Fatal("arbitrary request")
		}
	}
}
func TestUnavailableNegativeAndSupportScope(t *testing.T) {
	r, e := Parse("rpm", []byte("SEARCHING...\rNO DATA\r>"))
	if e != nil || len(r.Responses) != 0 || len(r.Statuses) != 2 {
		t.Fatal(r, e)
	}
	r, e = Parse("rpm", []byte("7E8 03 7F 01 12\r>"))
	if e != nil || r.Responses[0].State != "negative_response" {
		t.Fatal(r, e)
	}
	r, e = Parse("adapter_voltage", []byte("--.--\r>"))
	if e != nil || r.Responses[0].Value != nil {
		t.Fatal(r, e)
	}
	book := NewCapabilities("vehicle-A/adapter-1/generation-1")
	book.Unavailable("7E8", "rpm", "timeout")
	if book.Get("7E8", "rpm").State != "unknown" {
		t.Fatal("timeout inferred unsupported")
	}
	r, e = Parse("support", []byte("7E8 06 41 00 08 10 00 00\r7E9 06 41 00 00 00 00 00\r>"))
	if e != nil {
		t.Fatal(e)
	}
	for _, response := range r.Responses {
		if e = book.ApplySupport(response, "tx-1"); e != nil {
			t.Fatal(e)
		}
	}
	if book.Get("7E8", "rpm").State != "advertised" || book.Get("7E9", "rpm").State != "unsupported" {
		t.Fatal(book.Rows())
	}
	book.Observe("7E8", "rpm", "tx-2")
	book.Unavailable("7E8", "rpm", "tx-3")
	row := book.Get("7E8", "rpm")
	if row.State != "unavailable" || !row.Observed || !*row.Advertised {
		t.Fatal(row)
	}
	if NewCapabilities("vehicle-B").Get("7E8", "rpm").State != "unknown" {
		t.Fatal("foreign cache")
	}
}
func FuzzFramingAndParsing(f *testing.F) {
	f.Add([]byte("7E8 04 41 0C 1A F8\r>"))
	f.Add([]byte("NO DATA\r>"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 65537 {
			return
		}
		var framing Framer
		frames, _ := framing.Feed(data)
		for _, frame := range frames {
			_, _ = Parse("rpm", frame)
		}
	})
}

func TestPartialECUFailurePreservesOtherResponder(t *testing.T) {
	reply, err := Parse("rpm", []byte("7E8 04 41 0C 1A F8\r7E9 04 41 0C 00\r>"))
	if err != nil || len(reply.Responses) != 1 || len(reply.Failures) != 1 || reply.Failures[0].ECU != "7E9" {
		t.Fatal(reply, err)
	}
	book := NewCapabilities("scope")
	r, _ := Parse("support", []byte("7E8 06 41 00 08 10 00 00\r>"))
	_ = book.ApplySupport(r.Responses[0], "tx-1")
	row := book.Get("7E8", "rpm")
	*row.Advertised = false
	if !*book.Get("7E8", "rpm").Advertised {
		t.Fatal("shared capability pointer")
	}
}

func TestPromptWhitespaceAcrossEveryBoundary(t *testing.T) {
	data := []byte("\r\n010C\r7E8 04 41 0C 1A F8\r>\r\n")
	for split := 0; split <= len(data); split++ {
		var f Framer
		a, e := f.Feed(data[:split])
		if e != nil {
			t.Fatal(e)
		}
		b, e := f.Feed(data[split:])
		if e != nil {
			t.Fatal(e)
		}
		all := append(a, b...)
		if len(all) != 1 || f.Pending() != 0 {
			t.Fatal(split)
		}
		if _, e = Parse("rpm", all[0]); e != nil {
			t.Fatal(e)
		}
	}
}
