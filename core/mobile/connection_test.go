package mobile

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func capture(t *testing.T) *ConnectionCapture {
	t.Helper()
	c, e := NewConnectionCapture(t.TempDir(), `{"source":"fixture"}`, 1000, 100)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = c.Close("test", 10000) })
	return c
}
func TestConnectionTranscript(t *testing.T) {
	c := capture(t)
	if e := c.Event("streams_ready", "", 101); e != nil {
		t.Fatal(e)
	}
	data, e := c.IdentityRequest(102)
	if e != nil || string(data) != "STI\r" {
		t.Fatalf("%q %v", data, e)
	}
	if e = c.Accepted(data[:1], 103); e != nil {
		t.Fatal(e)
	}
	if e = c.Accepted(data[1:], 104); e != nil {
		t.Fatal(e)
	}
	for i, s := range []string{"STI\rSTN", "2232 v5.10.3\r", ">"} {
		if e = c.Receive([]byte(s), int64(105+i)); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = c.IdentityRequest(109); e == nil {
		t.Fatal("replayed")
	}
	if e = c.Close("disconnect", 110); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(c.Path())
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	for i, line := range lines {
		var row captureRow
		if e = json.Unmarshal([]byte(line), &row); e != nil || row.Sequence != int64(i+1) {
			t.Fatal(line, e)
		}
	}
	if len(lines) != 9 {
		t.Fatal(len(lines))
	}
}
func TestConnectionRejectsInvalidBytesAndStaleGeneration(t *testing.T) {
	c := capture(t)
	if e := c.Accepted([]byte("ATZ\r"), 101); e == nil {
		t.Fatal("arbitrary command")
	}
	if e := c.Receive(make([]byte, 65537), 101); e == nil {
		t.Fatal("oversized")
	}
	if e := c.Receive([]byte{0, 255}, 101); e != nil {
		t.Fatal(e)
	}
	if e := c.Receive([]byte{1}, 100); e == nil {
		t.Fatal("clock regression")
	}
	_, _ = c.IdentityRequest(102)
	_ = c.Accepted([]byte("S"), 103)
	if e := c.Close("lost", 104); e != nil {
		t.Fatal(e)
	}
	if e := c.Close("again", 105); e != nil {
		t.Fatal(e)
	}
	if e := c.Receive([]byte{1}, 106); e == nil {
		t.Fatal("closed accepted")
	}
	b, _ := os.ReadFile(c.Path())
	if !strings.Contains(string(b), "unsent=3") {
		t.Fatal(string(b))
	}
	next := capture(t)
	if next.Path() == c.Path() {
		t.Fatal("reused generation")
	}
}
func TestConnectionDiskFailureAndBudget(t *testing.T) {
	c := capture(t)
	c.size = (64 << 20) - 8192
	if e := c.Receive([]byte{1}, 101); e == nil {
		t.Fatal("budget ignored")
	}
	if e := c.Close("budget", 102); e != nil {
		t.Fatal(e)
	}
	d := capture(t)
	_ = d.file.Close()
	if e := d.Receive([]byte{1}, 101); e == nil {
		t.Fatal("disk failure hidden")
	}
	if e := d.Close("failed", 102); e == nil {
		t.Fatal("failure forgotten")
	}
}
