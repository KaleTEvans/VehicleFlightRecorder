package mobile

import (
	"fmt"
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
	if a.interruptedID != "" {
		t.Fatal("normal probe context mislabeled as interrupted drive")
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
	if b.previous == "" || b.interruptedID != b.previous || b.disk.ActiveDrive != "" {
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

func TestAutomaticDurationLimitDoesNotCreateChunks(t *testing.T) {
	a, e := NewAutomaticRecorder(t.TempDir(), "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	a.disk.Enabled = true
	if e = a.create("automatic", 1700000000000, fixtureNative); e != nil {
		t.Fatal(e)
	}
	a.policy.Recording = true
	first := a.live
	// Trigger the actual bound without pretending this is a sustained acquisition.
	if _, e = first.NextCommand(fixtureNative+1800*acquisition.Second, 1700001800000); e != nil {
		t.Fatal(e)
	}
	if e = a.Tick(fixtureNative+1800*acquisition.Second, 1700001800000); e != nil {
		t.Fatal(e)
	}
	if !a.policy.Suppressed || a.disk.ActiveDrive != "" || a.live.options.StartOrigin != "activity_probe" {
		t.Fatal("duration end restarted a drive")
	}
}
func TestExplicitAutomaticProfileAndInvalidClock(t *testing.T) {
	root := t.TempDir()
	a, e := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		t.Fatal(e)
	}
	if e = a.SetProfile(obd.Profile); e != nil {
		t.Fatal(e)
	}
	if e = a.SetEnabled(true, 1700000000000, fixtureNative); e != nil {
		t.Fatal(e)
	}
	if e = a.SetProfile(obd.CAN29Profile); e == nil {
		t.Fatal("armed profile changed")
	}
	if a.live.core.Session().Live.Protocol != obd.Profile {
		t.Fatal("selected profile not used")
	}
	a.Close()
	b, e := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	if b.profile != obd.Profile {
		t.Fatal("persisted profile was ignored")
	}
}

func TestClosedAutomaticOwnerCannotWriteAfterReplacement(t *testing.T) {
	root := t.TempDir()
	a, e := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		t.Fatal(e)
	}
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	b, e := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	if a.Live() != nil {
		t.Fatal("closed owner exposed live engine")
	}
	if e = a.Stop(fixtureNative, 1700000000000); e == nil {
		t.Fatal("closed stop wrote state")
	}
	if _, e = a.ClaimConfirmation(fixtureNative); e == nil {
		t.Fatal("closed claim wrote state")
	}
	if e = a.NativeEvent("foreground", "closed", fixtureNative); e == nil {
		t.Fatal("closed callback wrote journal")
	}
}

func TestFailedAutomaticDisarmExportAndRearmPreservesOriginals(t *testing.T) {
	root := t.TempDir()
	a, e := NewAutomaticRecorder(root, "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if e = a.SetEnabled(true, 1700000000000, fixtureNative); e != nil {
		t.Fatal(e)
	}
	old := a.Live()
	oldPath := old.Path()
	before, e := os.ReadFile(filepath.Join(oldPath, "session.json"))
	if e != nil {
		t.Fatal(e)
	}
	_ = a.fail(fmt.Errorf("adapter synchronization retries exhausted"))
	if e = a.SetEnabled(false, 1700000001000, fixtureNative+acquisition.Second); e != nil {
		t.Fatal("cannot disarm failed engine", e)
	}
	after, _ := os.ReadFile(filepath.Join(oldPath, "session.json"))
	if string(before) != string(after) {
		t.Fatal("failed original finalized/rewritten")
	}
	if a.disk.Enabled || a.live != nil || a.failure != "" {
		t.Fatal("failed owner still blocks diagnostic snapshot")
	}
	if e = a.SetEnabled(true, 1700000002000, fixtureNative+2*acquisition.Second); e != nil {
		t.Fatal(e)
	}
	if a.Live().Path() == oldPath {
		t.Fatal("rearm reused failed engine")
	}
}

// The native regression fixture must reproduce the real channel lifecycle:
// failed generations close before another EASession is created.
func TestAutomaticFourStartupTimeoutsCanDisarmAndRearm(t *testing.T) {
	a, err := NewAutomaticRecorder(t.TempDir(), "native-fixture", obd.CAN29Profile)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.SetEnabled(true, 1700000000000, fixtureNative); err != nil {
		t.Fatal(err)
	}
	l := a.Live()
	for attempt := int64(0); attempt < 4; attempt++ {
		now := fixtureNative + attempt*10*acquisition.Second
		gen, e := l.Connected(`{"protocol":"com.obdlink","name":"Scripted timeout fixture"}`, now)
		if e != nil {
			t.Fatal(attempt, e)
		}
		if e = l.NativeEvent(gen, now, "streams_ready", "scripted"); e != nil {
			t.Fatal(e)
		}
		data, e := l.NextCommand(now+acquisition.Second, 1700000000000+(attempt*10+1)*1000)
		if e != nil || string(data) != "STPC\r" {
			t.Fatal(attempt, string(data), e)
		}
		if e = l.Accepted(gen, now+acquisition.Second+1000, data); e != nil {
			t.Fatal(e)
		}
		_, _ = l.NextCommand(now+4*acquisition.Second, 1700000000000+(attempt*10+4)*1000)
		_, _ = l.NextCommand(now+6*acquisition.Second, 1700000000000+(attempt*10+6)*1000)
		if attempt < 3 {
			if e = l.Disconnected(gen, now+7*acquisition.Second, "scripted native close"); e != nil {
				t.Fatal(e)
			}
		}
	}
	if err = a.Tick(fixtureNative+40*acquisition.Second, 1700000040000); err == nil || !strings.Contains(err.Error(), "synchronization retries") {
		t.Fatal(err)
	}
	if a.disk.Enabled || a.state != "failed" {
		t.Fatal(a.Snapshot())
	}
	original := l.Path()
	if err = a.SetEnabled(false, 1700000041000, fixtureNative+41*acquisition.Second); err != nil {
		t.Fatal(err)
	}
	if err = a.SetEnabled(true, 1700000042000, fixtureNative+42*acquisition.Second); err != nil {
		t.Fatal(err)
	}
	if a.Live().Path() == original {
		t.Fatal("reused failed capture")
	}
}
