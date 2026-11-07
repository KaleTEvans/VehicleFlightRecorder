package mobile

import (
	"encoding/json"
	"testing"
	"vehicleflightrecorder.local/core/acquisition"
	"vehicleflightrecorder.local/core/obd"
)

func newRegularProfileFixture(t *testing.T, workload, profile string) *LiveRecorder {
	t.Helper()
	options, _ := json.Marshal(map[string]string{"origin": "scripted-fixture", "operating_state": "stationary_running", "workload": workload, "reference_note": "unmeasured scripted profile fixture", "protocol_profile": profile})
	l, err := NewLiveRecorder(t.TempDir(), string(options), 1700000000000, fixtureNative)
	if err != nil {
		t.Fatal(err)
	}
	return l
}
func TestRegularWorkloadFixedProfilesAndLegacyDefault(t *testing.T) {
	for _, profile := range []string{"", obd.Profile, obd.CAN29Profile} {
		for _, workload := range []string{"combined", "rpm_only", "coolant_only", "voltage_only"} {
			t.Run(profile+"/"+workload, func(t *testing.T) {
				l := newRegularProfileFixture(t, workload, profile)
				g := connectLive(t, l, 0)
				seen := map[string]bool{}
				for now := int64(0); now < 75*acquisition.Second; now += acquisition.Second / 10 {
					data, err := l.NextCommand(fixtureNative+now, 1700000000000+now/1000000)
					if err != nil {
						t.Fatal(err)
					}
					if len(data) == 0 {
						continue
					}
					text := string(data)
					seen[text] = true
					reply := liveReplies[text]
					if profile == obd.CAN29Profile {
						reply = can29Reply(text)
					}
					if reply == "" {
						t.Fatalf("unexpected request %q", text)
					}
					if err = l.Accepted(g, fixtureNative+now+1000, data); err != nil {
						t.Fatal(err)
					}
					if err = l.Receive(g, fixtureNative+now+2000, []byte(reply)); err != nil {
						t.Fatal(err)
					}
				}
				for command, want := range map[string]bool{"0100\r": workload != "voltage_only", "010C\r": workload == "combined" || workload == "rpm_only", "0105\r": workload == "combined" || workload == "coolant_only", "STVR\r": workload == "combined" || workload == "voltage_only", "ATCS\r": false} {
					if seen[command] != want {
						t.Fatalf("%s requested=%v wanted=%v", command, seen[command], want)
					}
				}
				wantProfile, wantInit := obd.Profile, obd.InitializationVersion
				if profile == obd.CAN29Profile {
					wantProfile, wantInit = obd.CAN29Profile, obd.CAN29RuntimeInitializationVersion
				}
				report := l.scheduler.Report()
				if report.Profile != wantProfile || report.Configuration.Initialization != wantInit || l.core.Session().Live.Protocol != wantProfile {
					t.Fatalf("crossed profile %+v", report.Configuration)
				}
				if len(report.Observations) == 0 {
					t.Fatal("missing requested observations")
				}
				if err := l.Stop(fixtureNative + 76*acquisition.Second); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestRegularCAN29ManualWindowAndReconnect(t *testing.T) {
	l := newRegularProfileFixture(t, "combined", obd.CAN29Profile)
	g := connectLive(t, l, 0)
	freshSupport := false
	for now := int64(0); now <= 280*acquisition.Second; now += acquisition.Second / 10 {
		if now == 80*acquisition.Second {
			if _, err := l.Mark("Scripted CAN29 incident", fixtureNative+now); err != nil {
				t.Fatal(err)
			}
		}
		if now == 220*acquisition.Second {
			old := g
			if err := l.Disconnected(g, fixtureNative+now, "scripted loss after full incident window"); err != nil {
				t.Fatal(err)
			}
			g = connectLive(t, l, now+1)
			freshSupport = false
			if len(l.cached) != 0 || len(l.scheduler.CapabilityRows()) != 0 {
				t.Fatal("stale cache survived reconnect")
			}
			if err := l.Receive(old, fixtureNative+now+2, []byte("18DAF110 04 41 0C 0C 80\r>")); err == nil {
				t.Fatal("old generation received")
			}
		}
		data, err := l.NextCommand(fixtureNative+now+3, 1700000000000+now/1000000)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 {
			continue
		}
		text := string(data)
		if (text == "010C\r" || text == "0105\r") && !freshSupport {
			t.Fatal("measurement before fresh support")
		}
		if text == "ATCS\r" || text == "STP33\r" || text == "ATSH7DF\r" {
			t.Fatalf("wrong regular request %q", text)
		}
		if err = l.Accepted(g, fixtureNative+now+1000, data); err != nil {
			t.Fatal(err)
		}
		if err = l.Receive(g, fixtureNative+now+2000, []byte(can29Reply(text))); err != nil {
			t.Fatal(err)
		}
		if text == "0100\r" {
			freshSupport = true
		}
	}
	if err := l.Stop(fixtureNative + 281*acquisition.Second); err != nil {
		t.Fatal(err)
	}
	incidents := l.core.Incidents()
	if len(incidents) != 1 || incidents[0].Status != "captured" {
		t.Fatalf("manual capture %+v", incidents)
	}
	if l.scheduler.Generation() != 2 || !freshSupport || l.core.Session().State != "complete" {
		t.Fatal("recovery not complete")
	}
	for _, row := range l.cached {
		if row.Generation != 2 {
			t.Fatal("old generation displayed fresh")
		}
	}
}
func TestExplicitProfileRejectsDiagnosticHandoffAndUnknown(t *testing.T) {
	for _, test := range []struct{ workload, profile string }{{"combined", "AUTO"}, {"combined", "ISO15765-4-CAN29-500"}, {obd.DiagnosticWorkload, obd.CAN29Profile}, {obd.CAN29Workload, obd.CAN29Profile}, {obd.HandoffWorkload, obd.CAN29Profile}} {
		options, _ := json.Marshal(map[string]string{"origin": "scripted-fixture", "operating_state": "stationary_running", "workload": test.workload, "protocol_profile": test.profile})
		if _, err := NewLiveRecorder(t.TempDir(), string(options), 1700000000000, fixtureNative); err == nil {
			t.Fatalf("accepted ambiguous profile %+v", test)
		}
	}
}
