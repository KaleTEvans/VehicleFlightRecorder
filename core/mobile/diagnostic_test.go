package mobile

import (
	"encoding/json"
	"strings"
	"testing"
	"vehicleflightrecorder.local/core/acquisition"
	"vehicleflightrecorder.local/core/obd"
)

func diagnosticReply(command string) string {
	switch command {
	case "ATSH7DF\r", "STCAF0\r", "ATCRA\r", "STFA\r":
		return "OK\r>"
	case "STPBRR\r":
		return "500000\r>"
	case "ATCS\r":
		return "T:00 R:00\r>"
	default:
		return liveReplies[command]
	}
}
func TestDiagnosticNoDataRecoveryAndGenerationIsolation(t *testing.T) {
	for _, recoverSupport := range []bool{false, true} {
		name := "no_data"
		if recoverSupport {
			name = "recovery"
		}
		t.Run(name, func(t *testing.T) {
			l := newLiveFixture(t, obd.DiagnosticWorkload)
			g := connectLive(t, l, 0)
			attempts := 0
			supportOK := false
			measurements := 0
			for now := int64(0); now <= 110*acquisition.Second; now += acquisition.Second / 10 {
				command, err := l.NextCommand(fixtureNative+now, 1700000000000+now/1000000)
				if err != nil {
					t.Fatal(err)
				}
				if len(command) == 0 {
					continue
				}
				text := string(command)
				reply := diagnosticReply(text)
				if text == "0100\r" {
					attempts++
					if !recoverSupport || attempts == 1 {
						reply = "NO DATA\r>"
					} else {
						supportOK = true
					}
				}
				if text == "010C\r" || text == "0105\r" {
					measurements++
					if !supportOK {
						t.Fatal("measurement before support")
					}
				}
				if reply == "" {
					t.Fatalf("unreviewed %q", text)
				}
				if err = l.Accepted(g, fixtureNative+now+1000, command); err != nil {
					t.Fatal(err)
				}
				if err = l.Receive(g, fixtureNative+now+2000, []byte(reply)); err != nil {
					t.Fatal(err)
				}
			}
			if attempts < 3 || !l.scheduler.Initialized() {
				t.Fatalf("attempts=%d setup=%v", attempts, l.scheduler.Initialized())
			}
			report := l.scheduler.Report()
			if report.Configuration.Initialization != obd.DiagnosticInitializationVersion {
				t.Fatal("wrong profile version")
			}
			if recoverSupport != (measurements > 0) {
				t.Fatalf("incorrect measurement gate %d", measurements)
			}
			if len(report.Observations) == 0 {
				t.Fatal("adapter voltage missing")
			}
			for _, o := range report.Observations {
				if !recoverSupport && o.Signal != "electrical.adapter_voltage" {
					t.Fatal("invented ECU reading")
				}
			}

			// Direct map checks preserve the public snapshot's snake-case contract.
			var value map[string]any
			if err := json.Unmarshal([]byte(l.Snapshot()), &value); err != nil {
				t.Fatal(err)
			}
			d := value["discovery"].(map[string]any)
			if d["attempts"].(float64) != float64(attempts) || d["setup_complete"].(float64) != 21 || d["can_status"] != "T:00 R:00" {
				t.Fatalf("snapshot %+v", d)
			}
			// Every support retry after NO DATA observes the scheduler's conservative 30s backoff.
			var prior *acquisition.Transaction
			for i := range report.Transactions {
				tx := &report.Transactions[i]
				if tx.Key != "support" {
					continue
				}
				if prior != nil && prior.Outcome == "no_data" && tx.StartNS-*prior.EndNS < 30*acquisition.Second {
					t.Fatal("no-data hot loop")
				}
				prior = tx
			}
			if err := l.Disconnected(g, fixtureNative+111*acquisition.Second, "fixture"); err != nil {
				t.Fatal(err)
			}
			g = connectLive(t, l, 112*acquisition.Second)
			command, err := l.NextCommand(fixtureNative+112*acquisition.Second, 1700000112000)
			if err != nil || string(command) != "STPC\r" {
				t.Fatalf("fresh setup %q %v", command, err)
			}
			var after map[string]any
			json.Unmarshal([]byte(l.Snapshot()), &after)
			reset := after["discovery"].(map[string]any)
			if reset["attempts"].(float64) != 0 || reset["can_status"] != "" {
				t.Fatal("foreign-generation status")
			}
			if err := l.Stop(fixtureNative + 113*acquisition.Second); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestDiagnosticBaudMismatchStopsBeforeVehicleRequests(t *testing.T) {
	l := newLiveFixture(t, obd.DiagnosticWorkload)
	g := connectLive(t, l, 0)
	for now := int64(0); now < 25*acquisition.Second; now += acquisition.Second / 10 {
		command, err := l.NextCommand(fixtureNative+now, 1700000000000+now/1000000)
		if err != nil {
			if !l.RequiresReconnect() {
				t.Fatal(err)
			}
			return
		}
		if len(command) == 0 {
			continue
		}
		text := string(command)
		if strings.HasPrefix(text, "01") {
			t.Fatal("vehicle request after bad baud")
		}
		reply := diagnosticReply(text)
		if text == "STPBRR\r" {
			reply = "250000\r>"
		}
		if err = l.Accepted(g, fixtureNative+now+1000, command); err != nil {
			t.Fatal(err)
		}
		if err = l.Receive(g, fixtureNative+now+2000, []byte(reply)); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("baud mismatch did not block")
}
