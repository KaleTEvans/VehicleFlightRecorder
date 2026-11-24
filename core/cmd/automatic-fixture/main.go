// automatic-fixture exercises the actual Go owner with scripted bytes, never a
// physical adapter. It produces retained probes plus drive and policy evidence.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"vehicleflightrecorder.local/core/mobile"
	"vehicleflightrecorder.local/core/obd"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	output := flag.String("output", "", "new output directory")
	scenario := flag.String("scenario", "running", "running, engine_off, manual_stop, interrupted, reconnect, startup_stopped")
	flag.Parse()
	if *output == "" {
		return fmt.Errorf("output required")
	}
	path, e := filepath.Abs(*output)
	if e != nil {
		return e
	}
	if e = os.Mkdir(path, 0700); e != nil {
		return e
	}
	a, e := mobile.NewAutomaticRecorder(path, "scripted-fixture", obd.CAN29Profile)
	if e != nil {
		return e
	}
	defer a.Close()
	const base int64 = 1000000000000
	const anchor int64 = 1700000000000
	if e = a.SetEnabled(true, anchor, base); e != nil {
		return e
	}
	var old *mobile.LiveRecorder
	var generation int64
	stopped := map[string]bool{}
	metadata := `{"protocol":"com.obdlink","name":"Explicit scripted M7 fixture"}`
	for step := int64(0); step <= 3000; step++ {
		now := step * 100000000
		if e = a.Tick(base+now, anchor+now/1000000); e != nil {
			return e
		}
		l := a.Live()
		if l == nil {
			continue
		}
		if l != old {
			generation, e = l.Connected(metadata, base+now)
			if e != nil {
				return e
			}
			if e = l.NativeEvent(generation, base+now, "streams_ready", "scripted fixture"); e != nil {
				return e
			}
			if *scenario == "startup_stopped" {
				if e = l.Receive(generation, base+now, []byte("residual old reply\r>")); e != nil {
					return e
				}
			}
			old = l
		}
		if *scenario == "manual_stop" && step == 900 {
			if e = a.Stop(base+now, anchor+now/1000000); e != nil {
				return e
			}
			continue
		}
		if *scenario == "reconnect" && step == 2200 {
			if e = l.Disconnected(generation, base+now, "scripted loss"); e != nil {
				return e
			}
			continue
		}
		if *scenario == "reconnect" && step > 2200 && step < 2300 {
			continue
		}
		if *scenario == "reconnect" && step == 2300 {
			generation, e = l.Connected(metadata, base+now)
			if e != nil {
				return e
			}
			if e = l.NativeEvent(generation, base+now, "streams_ready", "scripted recovery"); e != nil {
				return e
			}
		}
		markStep := int64(1000)
		if *scenario == "startup_stopped" {
			markStep = 1100
		}
		if step == markStep {
			var meta map[string]any
			if e = json.Unmarshal([]byte(l.Snapshot()), &meta); e != nil {
				return e
			}
			if meta["start_origin"] == "automatic" {
				if _, e = l.Mark("Scripted M7 manual window", base+now); e != nil {
					return e
				}
			}
		}
		bytes, e := l.NextCommand(base+now, anchor+now/1000000)
		if e != nil {
			return e
		}
		if len(bytes) == 0 {
			continue
		}
		request := string(bytes)
		reply := "OK\r>"
		switch request {
		case "STPR\r":
			reply = "34\r>"
		case "STPBRR\r":
			reply = "500000\r>"
		case "STI\r":
			reply = "STN2255 v5.8.1\r>"
		case "0100\r":
			reply = "18DAF110 06 41 00 08 10 00 00\r18DAF117 06 41 00 08 10 00 00\r>"
		case "010C\r":
			rpm := "0E D8"
			if *scenario == "engine_off" || step < 250 || (*scenario == "running" && step > 2300) {
				rpm = "00 00"
			}
			reply = "18DAF110 04 41 0C " + rpm + "\r18DAF117 04 41 0C " + rpm + "\r>"
		case "0105\r":
			reply = "18DAF110 03 41 05 78\r>"
		case "STVR\r":
			reply = "13.45\r>"
		default:
			found := false
			steps, _ := obd.InitializationFor(obd.CAN29RuntimeInitializationVersion)
			for _, s := range steps {
				if s.Command == request {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("unreviewed scripted request %q", request)
			}
		}
		if *scenario == "startup_stopped" && request == "STPC\r" && !stopped[l.Path()] {
			stopped[l.Path()] = true
			reply = "STPC\rSTOPPED\r\r>"
		}
		if e = l.Accepted(generation, base+now+1000, bytes); e != nil {
			return e
		}
		if e = l.Receive(generation, base+now+2000, []byte(reply)); e != nil {
			return e
		}
		if _, e = a.ClaimConfirmation(base + now + 3000); e != nil {
			return e
		}
	}
	if *scenario != "interrupted" {
		if e = a.SetEnabled(false, anchor+301000, base+301000000000); e != nil {
			return e
		}
	}
	fmt.Println(path)
	return nil
}
