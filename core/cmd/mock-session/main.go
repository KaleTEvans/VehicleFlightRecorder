// Command mock-session records a deterministic healthy drive without hardware.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"vehicleflightrecorder.local/core/mock"
	"vehicleflightrecorder.local/core/recorder"
	"vehicleflightrecorder.local/core/vehicle"
)

func run(args []string, stdout, stderr io.Writer) (result error) {
	flags := flag.NewFlagSet("mock-session", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("output", "", "new session directory (must not exist)")
	pack := flags.String("pack", "", "vehicle pack directory with explicitly mock-only signals")
	id := flags.String("session-id", "healthy-demo", "deterministic session identity")
	start := flags.String("start", "2026-10-23T12:00:00Z", "logical start time, RFC3339 with millisecond precision")
	seed := flags.Int64("seed", 42, "deterministic random seed")
	duration := flags.Duration("duration", 12*time.Minute, "logical duration, whole seconds, 2m to 24h")
	window := flags.Duration("buffer-window", time.Minute, "rolling retention, whole milliseconds")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *output == "" || *pack == "" || flags.NArg() != 0 {
		return fmt.Errorf("--output and --pack are required; unexpected positional arguments are not accepted")
	}
	timestamp, err := time.Parse(time.RFC3339Nano, *start)
	if err != nil {
		return fmt.Errorf("--start: %w", err)
	}
	if timestamp.Nanosecond()%int(time.Millisecond) != 0 || *duration%time.Second != 0 {
		return fmt.Errorf("start must have millisecond precision; duration must be whole seconds")
	}
	profile, err := vehicle.Load(*pack)
	if err != nil {
		return err
	}
	config := mock.Config{SessionID: *id, StartTimeUnixMS: timestamp.UnixMilli(), Seed: *seed, DurationMS: duration.Milliseconds()}
	session, err := config.Session(profile)
	if err != nil {
		return err
	}
	core, err := recorder.New(*window)
	if err != nil {
		return err
	}
	if err := core.Start(*output, session); err != nil {
		return err
	}
	defer func() { result = errors.Join(result, core.Close()) }()
	if err := mock.Healthy(context.Background(), config, profile, core.Append); err != nil {
		return err
	}
	if err := core.Stop(config.StartTimeUnixMS + config.DurationMS); err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(core.Session())
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "mock-session:", err)
		os.Exit(1)
	}
}
