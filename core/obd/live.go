package obd

import (
	"fmt"
	"regexp"
	"strings"
)

const LiveCatalogVersion = "standard-live-can11-1"
const InitializationVersion = "obdlink-stp33-runtime-1"
const DiagnosticInitializationVersion = "obdlink-stp33-diagnostic-2"
const DiagnosticWorkload = "discovery_diagnostic"
const HandoffWorkload = "adapter_handoff"
const HandoffInitializationVersion = "obdlink-adapter-handoff-1"
const HandoffProfile = "adapter-state-only"
const HandoffCatalog = "adapter-state-1"

var handoffInitialization = []Definition{
	{Key: "init_previous_protocol", Command: "STPR\r", Reference: STNReference + "#page=34"},
	{Key: "init_previous_description", Command: "STPRS\r", Reference: STNReference + "#page=34"},
	{Key: "init_previous_baud", Command: "STPBRR\r", Reference: STNReference + "#page=34"},
}

// Runtime-only configuration, FRPM E sections 7.2/8.6. No ATSP/NVM,
// calibration, ECU mutation, arbitrary CAN frame or reboot command is admitted.
var initialization = []Definition{
	{Key: "init_close", Command: "STPC\r", Reference: STNReference + "#page=34"},
	{Key: "init_defaults", Command: "ATD\r", Reference: STNReference + "#page=14"},
	{Key: "init_echo", Command: "ATE0\r", Reference: STNReference + "#page=15"},
	{Key: "init_linefeeds", Command: "ATL0\r", Reference: STNReference + "#page=16"},
	{Key: "init_memory", Command: "ATM0\r", Reference: STNReference + "#page=16"},
	{Key: "init_spaces", Command: "ATS1\r", Reference: STNReference + "#page=18"},
	{Key: "init_headers", Command: "ATH1\r", Reference: STNReference + "#page=15"},
	{Key: "init_dlc", Command: "ATD0\r", Reference: STNReference + "#page=15"},
	{Key: "init_format", Command: "ATCAF1\r", Reference: STNReference + "#page=13"},
	{Key: "init_flow", Command: "ATCFC1\r", Reference: STNReference + "#page=14"},
	{Key: "init_responses", Command: "ATR1\r", Reference: STNReference + "#page=17"},
	{Key: "init_timing", Command: "ATAT0\r", Reference: STNReference + "#page=12"},
	{Key: "init_protocol", Command: "STP33\r", Reference: STNReference + "#page=33"},
	{Key: "init_timeout", Command: "STPTO500\r", Reference: STNReference + "#page=34"},
	{Key: "init_protocol_check", Command: "STPR\r", Reference: STNReference + "#page=34"},
	{Key: "init_identity", Command: "STI\r", Reference: STNReference + "#page=28"},
}

func Initialization() []Definition { return append([]Definition(nil), initialization...) }

// Diagnostic setup preserves the legacy sequence and adds only fixed runtime
// addressing/filter settings and actual-baud readback after protocol selection.
func InitializationFor(version string) ([]Definition, error) {
	if version == HandoffInitializationVersion {
		return append([]Definition(nil), handoffInitialization...), nil
	}
	if version == InitializationVersion {
		return Initialization(), nil
	}
	if version != DiagnosticInitializationVersion {
		return nil, fmt.Errorf("unknown initialization version")
	}
	steps := Initialization()
	extra := []Definition{
		{Key: "init_functional_header", Command: "ATSH7DF\r", Reference: STNReference + "#page=18"},
		{Key: "init_normal_addressing", Command: "STCAF0\r", Reference: STNReference + "#page=37"},
		{Key: "init_receive_defaults", Command: "ATCRA\r", Reference: STNReference + "#page=14"},
		{Key: "init_automatic_filters", Command: "STFA\r", Reference: STNReference + "#page=39"},
		{Key: "init_baud_check", Command: "STPBRR\r", Reference: STNReference + "#page=34"},
	}
	// STP33 must precede settings whose interpretation depends on protocol.
	result := append([]Definition{}, steps[:13]...)
	result = append(result, extra...)
	return append(result, steps[13:]...), nil
}
func IsInitialization(key string) bool { return strings.HasPrefix(key, "init_") }
func LookupLive(key string) (Definition, error) {
	steps, _ := InitializationFor(DiagnosticInitializationVersion)
	steps = append(steps, handoffInitialization...)
	for _, d := range steps {
		if d.Key == key {
			return d, nil
		}
	}
	if key == "can_status" {
		return Definition{Key: key, Command: "ATCS\r", Reference: STNReference + "#page=14"}, nil
	}
	return Lookup(key)
}

var identityReply = regexp.MustCompile(`^STN[0-9]{4} v[0-9]+(?:\.[0-9]+){1,3}$`)

// ParseInitialization requires a complete, unambiguous acknowledgement. A bare
// prompt, wrong selected protocol, status/error text or extra line cannot enable
// vehicle requests. Echo is optional during configuration.
func ParseInitialization(key string, frame []byte) (string, error) {
	d, err := LookupLive(key)
	if err != nil || !IsInitialization(key) {
		return "", fmt.Errorf("unknown initialization step")
	}
	if len(frame) < 2 || len(frame) > 8192 || frame[len(frame)-1] != '>' || strings.Count(string(frame), ">") != 1 {
		return "", fmt.Errorf("one complete initialization reply required")
	}
	for _, b := range frame {
		if b != '\r' && b != '\n' && b != '\t' && (b < 32 || b > 126) {
			return "", fmt.Errorf("invalid initialization ASCII")
		}
	}
	var result []string
	for _, line := range strings.FieldsFunc(string(frame[:len(frame)-1]), func(r rune) bool { return r == '\r' || r == '\n' }) {
		line = strings.TrimSpace(line)
		if line == "" || compact(line) == compact(d.Command) {
			continue
		}
		result = append(result, line)
	}
	if strings.HasPrefix(key, "init_previous_") {
		if len(result) != 1 || len(result[0]) > 128 {
			return "", fmt.Errorf("ambiguous adapter handoff readback")
		}
		return result[0], nil
	}
	if len(result) != 1 {
		return "", fmt.Errorf("missing/ambiguous initialization acknowledgement")
	}
	want := "OK"
	if key == "init_protocol_check" {
		want = "33"
	}
	if key == "init_baud_check" {
		want = "500000"
	}
	if key == "init_identity" {
		if !identityReply.MatchString(result[0]) {
			return "", fmt.Errorf("unrecognized STN firmware identity")
		}
	} else if result[0] != want {
		return "", fmt.Errorf("initialization %s returned %q", key, result[0])
	}
	return result[0], nil
}

// ParseCANStatus retains a bounded adapter status reply, not a sensor observation
// or an assertion that the vehicle bus is healthy. Original bytes stay in raw.
func ParseCANStatus(frame []byte) (string, error) {
	if len(frame) < 2 || len(frame) > 256 || frame[len(frame)-1] != '>' || strings.Count(string(frame), ">") != 1 {
		return "", fmt.Errorf("one bounded CAN status reply required")
	}
	for _, b := range frame {
		if b != '\r' && b != '\n' && b != '\t' && (b < 32 || b > 126) {
			return "", fmt.Errorf("invalid status ASCII")
		}
	}
	lines := []string{}
	for _, line := range strings.FieldsFunc(string(frame[:len(frame)-1]), func(r rune) bool { return r == '\r' || r == '\n' }) {
		line = strings.TrimSpace(line)
		if line != "" && compact(line) != "ATCS" {
			lines = append(lines, line)
		}
	}
	if len(lines) != 1 || len(lines[0]) > 80 {
		return "", fmt.Errorf("ambiguous CAN status reply")
	}
	switch strings.ToUpper(lines[0]) {
	case "?", "OK", "NO DATA", "ERROR", "CAN ERROR", "BUS ERROR", "STOPPED":
		return "", fmt.Errorf("CAN status unavailable")
	}
	return lines[0], nil
}
