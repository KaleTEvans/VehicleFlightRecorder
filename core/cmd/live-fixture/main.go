// Live schema integration fixture. All bytes are scripted in-process; no I/O
// transport is opened and exported origin is always scripted-fixture.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"vehicleflightrecorder.local/core/mobile"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	output := flag.String("output", "", "new session parent directory")
	scenario := flag.String("scenario", "healthy", "healthy, reconnect, malformed, unsupported, timeout, no_data, recovery")
	workload := flag.String("workload", "combined", "combined, rpm_only, coolant_only, voltage_only, discovery_diagnostic")
	flag.Parse()
	if *output == "" {
		return fmt.Errorf("--output required")
	}
	root, e := filepath.Abs(*output)
	if e != nil {
		return e
	}
	valid := map[string]bool{"healthy": true, "reconnect": true, "malformed": true, "unsupported": true, "timeout": true, "no_data": true, "recovery": true, "handoff_retained": true, "handoff_unknown": true, "handoff_timeout": true}
	if !valid[*scenario] {
		return fmt.Errorf("unknown scripted scenario")
	}
	options, _ := json.Marshal(map[string]string{"origin": "scripted-fixture", "operating_state": "stationary_running", "workload": *workload, "reference_note": "Unmeasured scripted transcript; no physical reference comparison"})
	const anchor int64 = 1700000000000
	const base int64 = 1000000000000
	l, e := mobile.NewLiveRecorder(root, string(options), anchor, base)
	if e != nil {
		return e
	}
	metadata := `{"protocol":"com.obdlink","name":"Scripted fixture","source":"synthetic"}`
	gen, e := l.Connected(metadata, base)
	if e == nil {
		e = l.NativeEvent(gen, base, "streams_ready", "scripted fixture")
	}
	if e != nil {
		return e
	}
	replies := map[string]string{"STPC\r": "OK\r>", "ATD\r": "ATD\rOK\r>", "ATE0\r": "ATE0\rOK\r>", "ATL0\r": "OK\r>", "ATM0\r": "OK\r>", "ATS1\r": "OK\r>", "ATH1\r": "OK\r>", "ATD0\r": "OK\r>", "ATCAF1\r": "OK\r>", "ATCFC1\r": "OK\r>", "ATR1\r": "OK\r>", "ATAT0\r": "OK\r>", "STP33\r": "OK\r>", "STPTO500\r": "OK\r>", "STPR\r": "33\r>", "STI\r": "STN2255 v5.8.1\r>", "0100\r": "7E8 06 41 00 08 10 00 00\r7E9 06 41 00 00 10 00 00\r>", "010C\r": "7E8 04 41 0C 0C 80\r7E9 04 41 0C 0F A0\r>", "0105\r": "7E8 03 41 05 7B\r>", "STVR\r": "13.45\r>"}
	replies["ATSH7DF\r"] = "OK\r>"
	replies["STCAF0\r"] = "OK\r>"
	replies["ATCRA\r"] = "OK\r>"
	replies["STFA\r"] = "OK\r>"
	replies["STPBRR\r"] = "500000\r>"
	replies["ATCS\r"] = "T:00 R:00\r>"
	if *workload == "adapter_handoff" {
		replies["STPR\r"] = "34\r>"
		replies["STPRS\r"] = "ISO 15765-4 CAN 29/500\r>"
		if *scenario == "handoff_unknown" {
			replies["STPR\r"] = "AUTO\r>"
			replies["STPRS\r"] = "AUTO\r>"
			replies["STPBRR\r"] = "?\r>"
		}
	}
	supportAttempts := 0
	if *scenario == "unsupported" {
		replies["0100\r"] = "7E8 06 41 00 00 00 00 00\r>"
	}
	if *scenario == "malformed" {
		replies["010C\r"] = "7E8 04 41 0C 0C 80\r7E9 broken\r>"
	}
	lateAt := int64(-1)
	late := []byte(nil)
	timedOut := false
	offline := false
	for now := int64(0); now <= 210000000000; now += 100000000 {
		if *scenario == "reconnect" && now == 100000000000 {
			if e = l.Disconnected(gen, base+now, "scripted physical loss"); e != nil {
				return e
			}
			offline = true
		}
		if offline && now == 103000000000 {
			gen, e = l.Connected(metadata, base+now)
			if e == nil {
				e = l.NativeEvent(gen, base+now, "streams_ready", "scripted fixture")
			}
			if e != nil {
				return e
			}
			offline = false
		}
		if now == 80000000000 && *workload != "discovery_diagnostic" && *workload != "adapter_handoff" {
			if _, e = l.Mark("Scripted manual evidence", base+now); e != nil {
				return e
			}
		}
		if lateAt >= 0 && now >= lateAt {
			if e = l.Receive(gen, base+now, late); e != nil {
				return e
			}
			lateAt = -1
		}
		command, err := l.NextCommand(base+now, anchor+now/1000000)
		if err != nil {
			return err
		}
		if len(command) == 0 {
			continue
		}
		response, ok := replies[string(command)]
		if string(command) == "0100\r" {
			supportAttempts++
			if *scenario == "no_data" || (*scenario == "recovery" && supportAttempts == 1) {
				response = "NO DATA\r>"
			}
		}
		if !ok {
			return fmt.Errorf("unreviewed command %q", command)
		}
		if e = l.Accepted(gen, base+now+1000, command[:1]); e != nil {
			return e
		}
		if e = l.Accepted(gen, base+now+2000, command[1:]); e != nil {
			return e
		}
		if *scenario == "handoff_timeout" {
			continue
		}
		if *scenario == "timeout" && string(command) == "010C\r" && !timedOut {
			timedOut = true
			lateAt = now + 2100000000
			late = []byte(response)
			continue
		}
		half := len(response) / 2
		if e = l.Receive(gen, base+now+3000, []byte(response[:half])); e != nil {
			return e
		}
		if e = l.Receive(gen, base+now+4000, []byte(response[half:])); e != nil {
			return e
		}
	}
	if e = l.Stop(base + 211000000000); e != nil {
		return e
	}
	fmt.Println(l.Path())
	return nil
}
