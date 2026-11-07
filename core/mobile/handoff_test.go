package mobile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"vehicleflightrecorder.local/core/acquisition"
	"vehicleflightrecorder.local/core/obd"
)

func TestAdapterHandoffNeverConfiguresOrPollsVehicle(t *testing.T) {
	for _, kind := range []string{"retained", "unknown", "timeout", "malformed", "disconnect"} {
		t.Run(kind, func(t *testing.T) {
			l := newLiveFixture(t, obd.HandoffWorkload)
			gen := connectLive(t, l, 0)
			seen := []string{}
			expected := []string{"STPR\r", "STPRS\r", "STPBRR\r"}
			for now := int64(0); now <= 20*acquisition.Second; now += acquisition.Second / 10 {
				command, err := l.NextCommand(fixtureNative+now, 1700000000000+now/1000000)
				if err != nil {
					t.Fatal(err)
				}
				if len(command) == 0 {
					continue
				}
				if len(seen) >= 3 || string(command) != expected[len(seen)] {
					t.Fatalf("unexpected handoff write %q", command)
				}
				seen = append(seen, string(command))
				if kind == "disconnect" {
					if err = l.Disconnected(gen, fixtureNative+now+1000, "fixture handoff loss"); err != nil {
						t.Fatal(err)
					}
					continue
				}
				if err = l.Accepted(gen, fixtureNative+now+1000, command); err != nil {
					t.Fatal(err)
				}
				if kind == "timeout" {
					continue
				}
				reply := map[string]string{"STPR\r": "34\r>", "STPRS\r": "ISO 15765-4 CAN 29/500\r>", "STPBRR\r": "500000\r>"}[string(command)]
				if kind == "unknown" {
					reply = "?\r>"
				}
				if kind == "malformed" {
					reply = "34\rEXTRA\r>"
				}
				if err = l.Receive(gen, fixtureNative+now+2000, []byte(reply)); err != nil {
					t.Fatal(err)
				}
			}
			if l.core.State() != "complete" || l.RequiresReconnect() {
				t.Fatalf("handoff did not finalize without retry: %s", l.Snapshot())
			}
			report := l.scheduler.Report()
			if len(report.Observations) != 0 || len(report.Capabilities) != 0 || report.Profile != obd.HandoffProfile || report.Configuration.Protocol != "unmodified-adapter-state" {
				t.Fatal("handoff claimed vehicle acquisition")
			}
			if report.Configuration.MaximumDurationNS != handoffLimitNS {
				t.Fatal("unbounded capture")
			}
			if _, err := l.Mark("not a vehicle recording", fixtureNative+21*acquisition.Second); err == nil {
				t.Fatal("handoff incident marking allowed")
			}
			var snapshot map[string]any
			json.Unmarshal([]byte(l.Snapshot()), &snapshot)
			state := snapshot["adapter_state"].(map[string]any)
			expectedCount := float64(3)
			if kind == "timeout" || kind == "malformed" || kind == "disconnect" {
				expectedCount = 0
			}
			if state["captured"] != expectedCount {
				t.Fatalf("wrong readback count %+v", state)
			}
			if kind == "retained" && state["protocol"] != "34" {
				t.Fatal("previous setting overwritten")
			}
			if kind == "unknown" && state["protocol"] != "?" {
				t.Fatal("unknown setting invented")
			}
			var metadata map[string]any
			b, err := os.ReadFile(filepath.Join(l.Path(), "session.json"))
			if err != nil {
				t.Fatal(err)
			}
			json.Unmarshal(b, &metadata)
			if metadata["profile_id"] != obd.HandoffProfile || metadata["observation_count"] != float64(0) {
				t.Fatal("wrong session scope")
			}
		})
	}
}
func TestHandoffWithoutStreamsStillHasDurationBound(t *testing.T) {
	l := newLiveFixture(t, obd.HandoffWorkload)
	for now := int64(0); now <= 16*acquisition.Second; now += acquisition.Second {
		data, err := l.NextCommand(fixtureNative+now, 1700000000000+now/1000000)
		if err != nil || len(data) > 0 {
			t.Fatal("write without streams", err)
		}
	}
	if l.core.State() != "complete" {
		t.Fatal("handoff waiting indefinitely")
	}
}
