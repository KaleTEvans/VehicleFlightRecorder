package mobile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vehicleflightrecorder.local/core/acquisition"
	"vehicleflightrecorder.local/core/obd"
)

func TestAutomaticDriveClaimsOverrideAndRelaunch(t *testing.T) {
	root := t.TempDir()
	a, err := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, e := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile); e == nil {
		t.Fatal("second owner accepted")
	}
	if err = a.SetEnabled(true, 1700000000000, fixtureNative); err != nil {
		t.Fatal(err)
	}
	var old *LiveRecorder
	var g int64
	var drive string
	claims := 0
	for now := int64(0); now < 100*acquisition.Second; now += acquisition.Second / 10 {
		if err = a.Tick(fixtureNative+now, 1700000000000+now/1000000); err != nil {
			t.Fatal(err)
		}
		l := a.Live()
		if l != old {
			g = connectLive(t, l, now)
			old = l
		}
		data, e := l.NextCommand(fixtureNative+now, 1700000000000+now/1000000)
		if e != nil {
			t.Fatal(e)
		}
		if len(data) > 0 {
			reply := can29Reply(string(data))
			if now < 25*acquisition.Second && string(data) == "010C\r" {
				reply = "18DAF110 04 41 0C 00 00\r18DAF117 04 41 0C 00 00\r>"
			}
			if e = l.Accepted(g, fixtureNative+now+1000, data); e != nil {
				t.Fatal(e)
			}
			if e = l.Receive(g, fixtureNative+now+2000, []byte(reply)); e != nil {
				t.Fatal(e)
			}
		}
		if now < 25*acquisition.Second {
			entries, _ := os.ReadDir(filepath.Join(root, "Sessions"))
			if len(entries) > 0 {
				t.Fatal("engine-off created a drive")
			}
		}
		claim, e := a.ClaimConfirmation(fixtureNative + now + 3000)
		if e != nil {
			t.Fatal(e)
		}
		if claim != "" {
			claims++
			drive = a.disk.ActiveDrive
			if !strings.Contains(claim, "limited_standard_obd") {
				t.Fatal(claim)
			}
		}
	}
	if claims != 1 || drive == "" {
		t.Fatalf("claims=%d drive=%q", claims, drive)
	}
	if err = a.Stop(fixtureNative+101*acquisition.Second, 1700000101000); err != nil {
		t.Fatal(err)
	}
	if !a.disk.Suppressed {
		t.Fatal("stop not persisted")
	}
	// Relaunch cannot mutate originals or resend confirmations.
	path := filepath.Join(root, "Sessions", drive, "session.json")
	before, _ := os.ReadFile(path)
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	b, e := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	if !b.disk.Suppressed || !b.disk.Enabled {
		t.Fatal("override/preferences lost")
	}
	claim, e := b.ClaimConfirmation(fixtureNative + 102*acquisition.Second)
	if e != nil || claim != "" {
		t.Fatal(claim, e)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("original rewritten")
	}
}
func TestAutomaticInterruptedOriginalAndStorageFailure(t *testing.T) {
	root := t.TempDir()
	a, e := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		t.Fatal(e)
	}
	a.disk.ActiveDrive = "live-11111111111111111111111111111111"
	a.disk.Enabled = true
	if e = a.persist(); e != nil {
		t.Fatal(e)
	}
	a.Close()
	b, e := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	if b.previous == "" || b.disk.ActiveDrive != "" {
		t.Fatal("interrupted identity adopted")
	}
	if e = os.Remove(filepath.Join(root, "Automation", "state.json")); e != nil {
		t.Fatal(e)
	}
	if e = os.Mkdir(filepath.Join(root, "Automation", "state.json"), 0700); e != nil {
		t.Fatal(e)
	}
	if e = b.SetEnabled(false, 1700000000000, fixtureNative); e == nil {
		t.Fatal("storage failure hidden")
	}
	if b.state != "failed" {
		t.Fatal("failure not exposed")
	}
}
