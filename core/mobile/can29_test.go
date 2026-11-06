package mobile

import (
	"strings"
	"testing"
	"vehicleflightrecorder.local/core/acquisition"
	"vehicleflightrecorder.local/core/obd"
)

func can29Reply(command string) string {
	switch command {
	case "STP34\r", "ATSH18DB33F1\r":
		return "OK\r>"
	case "STPR\r":
		return "34\r>"
	default:
		return strings.NewReplacer("7E8", "18DAF110", "7E9", "18DAF118").Replace(diagnosticReply(command))
	}
}
func TestCAN29LiveDiscoveryAndReconnect(t *testing.T) {
	l := newLiveFixture(t, obd.CAN29Workload)
	g := connectLive(t, l, 0)
	attempts := 0
	seen := map[string]bool{}
	for now := int64(0); now <= 160*acquisition.Second; now += acquisition.Second / 10 {
		if now == 100*acquisition.Second {
			if err := l.Disconnected(g, fixtureNative+now, "scripted loss"); err != nil {
				t.Fatal(err)
			}
			g = connectLive(t, l, now+1)
		}
		command, err := l.NextCommand(fixtureNative+now+2, 1700000000000+now/1000000)
		if err != nil {
			t.Fatal(err)
		}
		if len(command) == 0 {
			continue
		}
		text := string(command)
		if text == "STP33\r" || text == "ATSH7DF\r" {
			t.Fatal("legacy addressing leaked")
		}
		reply := can29Reply(text)
		if text == "0100\r" {
			attempts++
			if attempts == 1 || g > 1 {
				reply = "NO DATA\r>"
			}
		}
		if text == "010C\r" || text == "0105\r" {
			if attempts < 2 || g > 1 {
				t.Fatal("measurement without fresh support")
			}
		}
		if reply == "" {
			t.Fatalf("unreviewed %q", text)
		}
		seen[text] = true
		if err = l.Accepted(g, fixtureNative+now+1000, command); err != nil {
			t.Fatal(err)
		}
		if err = l.Receive(g, fixtureNative+now+2000, []byte(reply)); err != nil {
			t.Fatal(err)
		}
	}
	report := l.scheduler.Report()
	if report.Profile != obd.CAN29Profile || report.Catalog != obd.CAN29Catalog || report.Configuration.Initialization != obd.CAN29InitializationVersion || report.Configuration.Protocol != "ISO15765-4-CAN29-500" {
		t.Fatalf("wrong profile %+v", report.Configuration)
	}
	if !seen["010C\r"] || !seen["0105\r"] || !seen["ATSH18DB33F1\r"] {
		t.Fatalf("missing requests %+v", seen)
	}
	found := map[string]bool{}
	for _, o := range report.Observations {
		if o.ECU != "adapter" {
			found[o.ECU] = true
			if o.Generation != 1 || !strings.HasPrefix(o.ECU, "18DAF1") {
				t.Fatalf("foreign observation %+v", o)
			}
		}
	}
	if !found["18DAF110"] || !found["18DAF118"] {
		t.Fatalf("lost full ECU IDs %+v", found)
	}
	for _, c := range report.Capabilities {
		if c.ECU != "adapter" {
			t.Fatalf("stale generation capability %+v", c)
		}
	}
	if err := l.Stop(fixtureNative + 161*acquisition.Second); err != nil {
		t.Fatal(err)
	}
}
func TestCAN29WrongReadbackBlocksVehicleRequests(t *testing.T) {
	for _, bad := range []struct{ command, reply string }{{"STPR\r", "33\r>"}, {"STPR\r", "A7 [AT]\r>"}, {"STPBRR\r", "250000\r>"}} {
		t.Run(bad.command+bad.reply, func(t *testing.T) {
			l := newLiveFixture(t, obd.CAN29Workload)
			g := connectLive(t, l, 0)
			failed := false
			for now := int64(0); now < 30*acquisition.Second; now += acquisition.Second / 10 {
				command, err := l.NextCommand(fixtureNative+now, 1700000000000+now/1000000)
				if err != nil {
					failed = true
					break
				}
				if len(command) == 0 {
					continue
				}
				text := string(command)
				if strings.HasPrefix(text, "01") {
					t.Fatal("vehicle request after bad readback")
				}
				reply := can29Reply(text)
				if text == bad.command {
					reply = bad.reply
				}
				if err = l.Accepted(g, fixtureNative+now+1000, command); err != nil {
					t.Fatal(err)
				}
				if err = l.Receive(g, fixtureNative+now+2000, []byte(reply)); err != nil {
					failed = true
					break
				}
			}
			if !failed || l.scheduler.Initialized() {
				t.Fatal("bad setup did not block")
			}
		})
	}
}
