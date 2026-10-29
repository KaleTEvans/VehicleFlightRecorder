// offline-acquisition has no native, serial, Bluetooth or socket dependencies.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"vehicleflightrecorder.local/core/acquisition"
)

func run(args []string) error {
	flags := flag.NewFlagSet("offline-acquisition", flag.ContinueOnError)
	output := flags.String("output", "", "new report file; existing files are never overwritten")
	scenario := flags.String("scenario", "healthy", "healthy or edge_cases; all data is synthetic")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *output == "" || flags.NArg() != 0 || (*scenario != "healthy" && *scenario != "edge_cases") {
		return fmt.Errorf("provide --output and --scenario healthy|edge_cases")
	}
	second := acquisition.Second
	scheduler, err := acquisition.New(1793000000000, []acquisition.Schedule{{Key: "support", IntervalNS: 10 * second, TimeoutNS: second / 2, Priority: 0}, {Key: "rpm", IntervalNS: second / 2, TimeoutNS: second / 2, Priority: 1}, {Key: "coolant", IntervalNS: second, TimeoutNS: second / 2, Priority: 2}, {Key: "adapter_voltage", IntervalNS: second, TimeoutNS: second / 2, Priority: 3}})
	if err != nil {
		return err
	}
	var work *acquisition.Work
	phase := 0
	sent := int64(0)
	timedOut, reconnected := false, false
	duration := 12 * second
	for now := int64(0); now < duration; now += second / 20 {
		if err = scheduler.Tick(now); err != nil {
			return err
		}
		if work == nil {
			work, err = scheduler.Poll(now)
			if err != nil {
				return err
			}
			phase = 0
			continue
		}
		switch phase {
		case 0:
			err = scheduler.Accepted(work.Generation, work.ID, now, 1)
			phase = 1
		case 1:
			if *scenario == "edge_cases" && work.Key == "adapter_voltage" && !reconnected {
				err = scheduler.Reconnect(now)
				reconnected = true
				work = nil
				continue
			}
			err = scheduler.Accepted(work.Generation, work.ID, now, len(work.Bytes)-1)
			sent = now
			phase = 2
		case 2:
			if *scenario == "edge_cases" && work.Key == "rpm" && !timedOut && now < sent+second/2+second/10 {
				continue
			}
			text := ""
			switch work.Key {
			case "support":
				text = "7E8 06 41 00 08 10 00 00\r7E9 06 41 00 00 10 00 00\r>"
			case "rpm":
				text = "7E8 04 41 0C 1A F8\r7E9 04 41 0C 00 00\r>"
			case "coolant":
				text = "7E8 03 41 05 7B\r>"
			case "adapter_voltage":
				text = "12.34\r>"
			}
			if *scenario == "edge_cases" && work.Key == "rpm" && !timedOut {
				timedOut = true
			}
			// Real framing code receives an echo and a split body in the same clock tick.
			body := string(work.Bytes) + text
			err = scheduler.Receive(work.Generation, now, []byte(body[:len(body)/2]))
			if err == nil {
				err = scheduler.Receive(work.Generation, now, []byte(body[len(body)/2:]))
			}
			work = nil
		}
		if err != nil {
			return err
		}
	}
	if err = scheduler.Stop(duration); err != nil {
		return err
	}
	report := scheduler.Report()
	if err = scheduler.WriteReport(*output); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"source": "synthetic", "state": report.State, "transactions": len(report.Transactions), "observations": len(report.Observations), "generation": report.Generation})
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
