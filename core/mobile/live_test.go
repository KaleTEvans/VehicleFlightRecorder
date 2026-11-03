package mobile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vehicleflightrecorder.local/core/acquisition"
	"vehicleflightrecorder.local/core/obd"
)

var liveReplies = map[string]string{
	"STPC\r": "OK\r>", "ATD\r": "ATD\rOK\r>", "ATE0\r": "ATE0\rOK\r>", "ATL0\r": "OK\r>", "ATM0\r": "OK\r>", "ATS1\r": "OK\r>", "ATH1\r": "OK\r>", "ATD0\r": "OK\r>", "ATCAF1\r": "OK\r>", "ATCFC1\r": "OK\r>", "ATR1\r": "OK\r>", "ATAT0\r": "OK\r>", "STP33\r": "OK\r>", "STPTO500\r": "OK\r>", "STPR\r": "33\r>", "STI\r": "STN2255 v5.8.1\r>",
	"0100\r": "7E8 06 41 00 08 10 00 00\r7E9 06 41 00 00 10 00 00\r>",
	"010C\r": "7E8 04 41 0C 0C 80\r7E9 04 41 0C 0F A0\r>", "0105\r": "7E8 03 41 05 7B\r>", "STVR\r": "13.45\r>",
}

const fixtureNative int64 = 1000000000000

func newLiveFixture(t *testing.T, workload string) *LiveRecorder {
	t.Helper()
	l, e := NewLiveRecorder(t.TempDir(), `{"origin":"scripted-fixture","operating_state":"stationary_running","workload":"`+workload+`","reference_note":"unmeasured synthetic transcript"}`, 1700000000000, fixtureNative)
	if e != nil {
		t.Fatal(e)
	}
	return l
}
func connectLive(t *testing.T, l *LiveRecorder, ns int64) int64 {
	t.Helper()
	g, e := l.Connected(`{"protocol":"com.obdlink","name":"Synthetic adapter","source":"fixture"}`, fixtureNative+ns)
	if e != nil {
		t.Fatal(e)
	}
	if e := l.NativeEvent(g, fixtureNative+ns, "streams_ready", "fixture"); e != nil {
		t.Fatal(e)
	}
	return g
}
func replyLive(t *testing.T, l *LiveRecorder, g, now int64, command []byte) {
	t.Helper()
	response, ok := liveReplies[string(command)]
	if !ok {
		t.Fatalf("unreviewed command %q", command)
	}
	if e := l.Accepted(g, fixtureNative+now+1000, command); e != nil {
		t.Fatal(e)
	}
	if e := l.Receive(g, fixtureNative+now+2000, []byte(response[:1])); e != nil {
		t.Fatal(e)
	}
	if e := l.Receive(g, fixtureNative+now+3000, []byte(response[1:])); e != nil {
		t.Fatal(e)
	}
}
func TestLiveInitializationManualIncidentAndAuthoritativeExport(t *testing.T) {
	l := newLiveFixture(t, "combined")
	g := connectLive(t, l, 0)
	initCount := 0
	for ns := int64(0); ns <= 210000000000; ns += 100000000 {
		if ns == 80000000000 {
			if _, e := l.Mark("fixture parked marker", fixtureNative+ns); e != nil {
				t.Fatal(e)
			}
		}
		data, e := l.NextCommand(fixtureNative+ns, 1700000000000+ns/1000000)
		if e != nil {
			t.Fatal(e)
		}
		if len(data) > 0 {
			if initCount < len(obd.Initialization()) {
				if string(data) != obd.Initialization()[initCount].Command {
					t.Fatalf("initialization gate %d got %q", initCount, data)
				}
				initCount++
			}
			replyLive(t, l, g, ns, data)
		}
	}
	if e := l.Stop(fixtureNative + 211000000000); e != nil {
		t.Fatal(e)
	}
	var meta map[string]any
	b, e := os.ReadFile(filepath.Join(l.Path(), "session.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &meta); e != nil {
		t.Fatal(e)
	}
	if meta["schema_version"] != float64(2) || meta["source"] != "obd-live" || meta["state"] != "complete" || meta["incident_count"] != float64(1) || meta["observation_count"].(float64) < 100 {
		t.Fatalf("metadata %+v", meta)
	}
	if len(l.core.Incidents()) != 1 || l.core.Incidents()[0].Status != "captured" {
		t.Fatalf("manual evidence: %+v", l.core.Incidents())
	}
	raw, e := os.ReadFile(filepath.Join(l.Path(), "raw/transport.jsonl"))
	if e != nil || len(raw) == 0 {
		t.Fatalf("raw evidence %v", e)
	}
	report := l.scheduler.Report()
	if report.Version != 2 || report.Source != "obd-live" || report.Configuration.Origin != "scripted-fixture" {
		t.Fatal("fixture mislabeled")
	}
	if strings.Contains(string(raw), "ATSP") || strings.Contains(string(raw), "ATZ") {
		t.Fatal("persistent protocol/reboot command leaked")
	}
	for _, o := range l.core.Snapshot() {
		if o.Observation != nil && o.Observation.Provenance == nil {
			t.Fatal("unreferenced value")
		}
	}
	t.Logf("session=%s observations=%v transactions=%d", l.Path(), meta["observation_count"], len(report.Transactions))
}
func TestLiveRejectsWrongInitializationAndNeverPollsVehicle(t *testing.T) {
	l := newLiveFixture(t, "combined")
	g := connectLive(t, l, 0)
	for ns := int64(0); ns < 20000000000; ns += 100000000 {
		data, e := l.NextCommand(fixtureNative+ns, 1700000000000+ns/1000000)
		if e != nil {
			if !l.RequiresReconnect() {
				t.Fatal(e)
			}
			return
		}
		if len(data) == 0 {
			continue
		}
		if string(data) == "0100\r" {
			t.Fatal("vehicle request before protocol validation")
		}
		if string(data) == "STPR\r" {
			if e = l.Accepted(g, fixtureNative+ns+1000, data); e != nil {
				t.Fatal(e)
			}
			if e = l.Receive(g, fixtureNative+ns+2000, []byte("34\r>")); e != nil {
				t.Fatal(e)
			}
		} else {
			replyLive(t, l, g, ns, data)
		}
	}
	t.Fatal("unsupported protocol did not block")
}
func TestLiveDisconnectCancelsPartialBytesAndReinitializes(t *testing.T) {
	l := newLiveFixture(t, "combined")
	old := connectLive(t, l, 0)
	data, e := l.NextCommand(fixtureNative, 1700000000000)
	if e != nil {
		t.Fatal(e)
	}
	if e = l.Accepted(old, fixtureNative+1000, data[:1]); e != nil {
		t.Fatal(e)
	}
	if e = l.Disconnected(old, fixtureNative+2000, "fixture disconnect"); e != nil {
		t.Fatal(e)
	}
	next := connectLive(t, l, 3000)
	if next == old {
		t.Fatal("generation reused")
	}
	if e = l.Receive(old, fixtureNative+4000, []byte("OK\r>")); e == nil {
		t.Fatal("stale receive admitted")
	}
	data, e = l.NextCommand(fixtureNative+5000, 1700000000000)
	if e != nil || string(data) != "STPC\r" {
		t.Fatalf("fresh setup %q %v", data, e)
	}
	r := l.scheduler.Report()
	if r.Transactions[0].Outcome != "cancelled" || r.Transactions[0].AcceptedBytes != 1 || r.Transactions[1].AcceptedBytes != 0 {
		t.Fatal("partial replay or lost cancellation")
	}
}
func TestLiveSnapshotDoesNotAcquireAndClockChangesAreEvidence(t *testing.T) {
	l := newLiveFixture(t, "voltage_only")
	connectLive(t, l, 0)
	for i := 0; i < 10; i++ {
		_ = l.Snapshot()
	}
	if len(l.scheduler.Report().Transactions) != 0 {
		t.Fatal("snapshot polled")
	}
	if _, e := l.NextCommand(fixtureNative+1000000000, 1700000050000); e != nil {
		t.Fatal(e)
	}
	records := l.core.Snapshot()
	found := false
	for _, r := range records {
		if r.Event != nil && r.Event.EventType == "wall_clock_adjusted" {
			found = true
			if r.Timestamp() != 1700000001000 {
				t.Fatal("wall clock rewrote mapping")
			}
		}
	}
	if !found {
		t.Fatal("clock adjustment not recorded")
	}
	if _, e := l.NextCommand(fixtureNative, 1700000000000); e == nil {
		t.Fatal("native clock regression admitted")
	}
}

func TestLiveBridgeCommandEnvelopePreservesIdleAndExactBytes(t *testing.T) {
	l := newLiveFixture(t, "voltage_only")
	encoded, err := l.NextCommandBase64(fixtureNative, 1700000000000)
	if err != nil || encoded != "" {
		t.Fatalf("idle envelope %q %v", encoded, err)
	}
	connectLive(t, l, 0)
	encoded, err = l.NextCommandBase64(fixtureNative, 1700000000000)
	if err != nil || encoded != "U1RQQw0=" {
		t.Fatalf("command envelope %q %v", encoded, err)
	}
}

func TestLiveRawPreservationFailureStopsWithoutRecovery(t *testing.T) {
	l := newLiveFixture(t, "voltage_only")
	connectLive(t, l, 0)
	l.scheduler.SetRawSink(func(_ acquisition.Raw) error { return fmt.Errorf("injected full disk") })
	data, err := l.NextCommand(fixtureNative, 1700000000000)
	if err == nil || len(data) != 0 || l.RequiresReconnect() || !l.failed {
		t.Fatalf("failed evidence must not send/reconnect: %q %v", data, err)
	}
	if _, err = l.NextCommand(fixtureNative+1000000000, 1700000001000); err == nil {
		t.Fatal("failed session resumed")
	}
	if err = l.Stop(fixtureNative + 1000000000); err == nil {
		t.Fatal("failed session finalized")
	}
}

func TestLiveEmptySnapshotCollectionsAreArraysForNativeDecoder(t *testing.T) {
	l := newLiveFixture(t, "combined")
	var snapshot map[string]json.RawMessage
	if err := json.Unmarshal([]byte(l.Snapshot()), &snapshot); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"readings", "capabilities", "incidents"} {
		if string(snapshot[key]) != "[]" {
			t.Fatalf("%s must be an empty array, got %s", key, snapshot[key])
		}
	}
}
