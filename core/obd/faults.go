package obd

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// FaultQuery is a fixed, read-only J1979 request. Freeze-frame queries refer
// to frame zero and return raw parameter bytes for that snapshot only.
type FaultQuery string

const FaultScanWorkload = "standard_fault_scan"

const (
	MonitorStatus  FaultQuery = "monitor_status"
	ConfirmedCodes FaultQuery = "confirmed_codes"
	PendingCodes   FaultQuery = "pending_codes"
	PermanentCodes FaultQuery = "permanent_codes"
	FreezeFrameDTC FaultQuery = "freeze_frame_dtc"
	FreezeFrameRPM FaultQuery = "freeze_frame_rpm"
	FreezeFrameECT FaultQuery = "freeze_frame_coolant"
)

// FaultScanQueries returns a fresh copy of the fixed scan order.
func FaultScanQueries() []FaultQuery {
	return []FaultQuery{MonitorStatus, ConfirmedCodes, PendingCodes, PermanentCodes, FreezeFrameDTC, FreezeFrameRPM, FreezeFrameECT}
}

type faultRequest struct {
	command string
	prefix  []byte
}

var faultRequests = map[FaultQuery]faultRequest{
	MonitorStatus:  {"0101\r", []byte{0x41, 0x01}},
	ConfirmedCodes: {"03\r", []byte{0x43}},
	PendingCodes:   {"07\r", []byte{0x47}},
	PermanentCodes: {"0A\r", []byte{0x4a}},
	FreezeFrameDTC: {"020200\r", []byte{0x42, 0x02, 0x00}},
	FreezeFrameRPM: {"020C00\r", []byte{0x42, 0x0c, 0x00}},
	FreezeFrameECT: {"020500\r", []byte{0x42, 0x05, 0x00}},
}

func (q FaultQuery) Command() (string, bool) {
	request, ok := faultRequests[q]
	return request.command, ok
}

type FaultResponse struct {
	ECU       string   `json:"ecu"`
	Outcome   string   `json:"outcome"`
	Codes     []string `json:"codes,omitempty"`
	MIL       *bool    `json:"mil,omitempty"`
	CodeCount *int     `json:"code_count,omitempty"`
	Data      []byte   `json:"data_base64,omitempty"`
}

type FaultReply struct {
	Outcome    string
	Responders []FaultResponse
	Failures   []Failure
}

type faultPayload struct {
	size, sequence int
	data           []byte
}

// ParseFaultReply expects the fixed header/PCI adapter configuration used in
// live captures. It reassembles each responder independently within one prompt.
func ParseFaultReply(profile string, query FaultQuery, frame []byte) (FaultReply, error) {
	request, ok := faultRequests[query]
	if !ok || (profile != Profile && profile != CAN29Profile) {
		return FaultReply{}, fmt.Errorf("unsupported fault query/profile")
	}
	if len(frame) == 0 || len(frame) > 8192 || frame[len(frame)-1] != '>' || strings.Count(string(frame), ">") != 1 {
		return FaultReply{}, fmt.Errorf("one complete fault response prompt required")
	}
	for _, b := range frame {
		if (b < 32 || b > 126) && b != '\r' && b != '\n' && b != '\t' {
			return FaultReply{}, fmt.Errorf("invalid fault response ASCII")
		}
	}
	lines := strings.FieldsFunc(string(frame[:len(frame)-1]), func(r rune) bool { return r == '\r' || r == '\n' })
	if len(lines) > 256 {
		return FaultReply{}, fmt.Errorf("too many fault response lines")
	}
	result := FaultReply{Outcome: "unavailable"}
	payloads := make(map[string]*faultPayload)
	bad := make(map[string]bool)
	var order []string
	statuses := []string{}
	for _, line := range lines {
		line = compact(line)
		switch line {
		case "", compact(request.command), "SEARCHING...":
			continue
		case "NODATA", "STOPPED", "?", "CANERROR", "BUSERROR", "UNABLETOCONNECT":
			statuses = append(statuses, line)
			continue
		}
		ecu, data, err := faultCANFrame(profile, line)
		if err == nil {
			if bad[ecu] {
				continue
			}
			if payloads[ecu] == nil {
				order = append(order, ecu)
				payloads[ecu] = &faultPayload{}
			}
			err = payloads[ecu].appendFrame(data)
		}
		if err != nil {
			bad[ecu] = true
			result.Failures = append(result.Failures, Failure{ecu, err.Error()})
		}
	}
	for _, ecu := range order {
		if bad[ecu] {
			continue
		}
		payload := payloads[ecu]
		response, err := decodeFaultPayload(query, request.prefix, ecu, payload.data)
		if len(payload.data) != payload.size {
			err = fmt.Errorf("incomplete ISO-TP reply")
		}
		if err != nil {
			result.Failures = append(result.Failures, Failure{ecu, err.Error()})
			continue
		}
		result.Responders = append(result.Responders, response)
	}
	positive := 0
	for _, row := range result.Responders {
		if row.Outcome == "reply" || row.Outcome == "no_codes" || row.Outcome == "no_freeze_frame_dtc" {
			positive++
		}
	}
	switch {
	case positive > 0 && (positive != len(result.Responders) || len(result.Failures) > 0 || len(statuses) > 0):
		result.Outcome = "partial"
	case positive > 0:
		result.Outcome = "ok"
	case len(result.Responders) > 0 && len(result.Failures) == 0 && len(statuses) == 0:
		result.Outcome = "negative_response"
	case len(result.Responders) == 0 && len(result.Failures) == 0 && len(statuses) == 1 && statuses[0] == "NODATA":
		result.Outcome = "no_data"
	}
	return result, nil
}

func faultCANFrame(profile, line string) (string, []byte, error) {
	width := 3
	if profile == CAN29Profile {
		width = 8
	}
	if len(line) < width+2 {
		return "unknown", nil, fmt.Errorf("missing CAN header/PCI")
	}
	ecu := line[:width]
	id, err := strconv.ParseUint(ecu, 16, 29)
	if err != nil || (profile == CAN29Profile && !validCAN29ID(ecu)) ||
		(profile == Profile && (id < 0x7e8 || id > 0x7ef)) {
		return "unknown", nil, fmt.Errorf("foreign CAN responder")
	}
	data, err := hex.DecodeString(line[width:])
	if err != nil || len(data) < 2 || len(data) > 8 {
		return ecu, nil, fmt.Errorf("invalid CAN payload")
	}
	return ecu, data, nil
}

func (p *faultPayload) appendFrame(data []byte) error {
	switch {
	case p.size == 0 && data[0]>>4 == 0:
		p.size = int(data[0])
		if p.size == 0 || p.size > len(data)-1 {
			return fmt.Errorf("invalid ISO-TP single-frame length")
		}
		p.data = append(p.data, data[1:1+p.size]...)
	case p.size == 0 && data[0]>>4 == 1:
		p.size = int(data[0]&15)<<8 | int(data[1])
		if len(data) != 8 || p.size < 8 || p.size > 512 {
			return fmt.Errorf("invalid ISO-TP first-frame length")
		}
		p.data = append(p.data, data[2:]...)
		p.sequence = 1
	case p.size > len(p.data) && data[0]>>4 == 2 && int(data[0]&15) == p.sequence:
		remaining := p.size - len(p.data)
		if remaining > 7 && len(data) != 8 || remaining <= 7 && len(data)-1 < remaining {
			return fmt.Errorf("truncated ISO-TP consecutive frame")
		}
		p.data = append(p.data, data[1:1+min(remaining, len(data)-1)]...)
		p.sequence = (p.sequence + 1) & 15
	default:
		return fmt.Errorf("unexpected ISO-TP frame or sequence")
	}
	return nil
}

func decodeFaultPayload(query FaultQuery, prefix []byte, ecu string, payload []byte) (FaultResponse, error) {
	response := FaultResponse{ECU: ecu, Outcome: "reply"}
	if len(payload) == 3 && payload[0] == 0x7f && payload[1] == prefix[0]-0x40 {
		response.Outcome, response.Data = "negative_response", payload[2:]
		return response, nil
	}
	if len(payload) < len(prefix) || string(payload[:len(prefix)]) != string(prefix) {
		return response, fmt.Errorf("foreign service, PID or freeze-frame number")
	}
	values := payload[len(prefix):]
	response.Data = values
	switch query {
	case MonitorStatus:
		if len(values) != 4 {
			return response, fmt.Errorf("invalid monitor status length")
		}
		mil, count := values[0]&0x80 != 0, int(values[0]&0x7f)
		response.MIL, response.CodeCount = &mil, &count
	case ConfirmedCodes, PendingCodes, PermanentCodes:
		if len(values) == 0 || len(values) != 1+2*int(values[0]) {
			return response, fmt.Errorf("DTC count does not match payload")
		}
		response.Codes = make([]string, 0, int(values[0]))
		if values[0] == 0 {
			response.Outcome = "no_codes"
		}
		for i := 1; i < len(values); i += 2 {
			if values[i] == 0 && values[i+1] == 0 {
				return response, fmt.Errorf("empty code in reported DTC list")
			}
			response.Codes = append(response.Codes, formatDTC(values[i], values[i+1]))
		}
	case FreezeFrameDTC:
		if len(values) != 2 {
			return response, fmt.Errorf("invalid freeze-frame code length")
		}
		if values[0] == 0 && values[1] == 0 {
			response.Outcome = "no_freeze_frame_dtc"
		} else {
			response.Codes = []string{formatDTC(values[0], values[1])}
		}
	case FreezeFrameRPM, FreezeFrameECT:
		width := 1
		if query == FreezeFrameRPM {
			width = 2
		}
		if len(values) != width {
			return response, fmt.Errorf("invalid freeze-frame parameter length")
		}
	}
	return response, nil
}

func formatDTC(high, low byte) string {
	return fmt.Sprintf("%c%X%X%02X", "PCBU"[high>>6], (high>>4)&3, high&15, low)
}
