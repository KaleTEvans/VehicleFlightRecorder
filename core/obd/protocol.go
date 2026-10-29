// Package obd implements an offline-only, deliberately narrow ASCII/CAN profile.
// It is not imported by the mobile facade and cannot open or write a transport.
package obd

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const CatalogVersion = "standard-offline-1"
const DecoderVersion = "1"
const Profile = "can11-single-frame-headers-pci"
const ELMReference = "https://www.elmelectronics.com/wp-content/uploads/2016/07/ELM327DS.pdf"
const STNReference = "https://www.scantool.net/scantool/downloads/678/obdlink_frpm_e.pdf"

// Definition is copied from the fixed catalog; callers cannot inject commands.
type Definition struct {
	Key, Command, Signal, Unit, Reference string
	PID                                   byte
	Bytes                                 int
}

var catalog = []Definition{
	{"support", "0100\r", "", "", ELMReference + "#page=31", 0, 4},
	{"rpm", "010C\r", "powertrain.engine.rpm", "rpm", ELMReference + "#page=31", 0x0c, 2},
	{"coolant", "0105\r", "powertrain.coolant.temperature", "degC", ELMReference + "#page=31", 5, 1},
	{"adapter_voltage", "STVR\r", "electrical.adapter_voltage", "V", STNReference + "#page=32", 0, 0},
}

func Catalog() []Definition { return append([]Definition(nil), catalog...) }
func Lookup(key string) (Definition, error) {
	for _, d := range catalog {
		if d.Key == key {
			return d, nil
		}
	}
	return Definition{}, fmt.Errorf("unknown/disabled request %q", key)
}

// Framer retains at most 8 KiB, accepts <=64 KiB chunks and emits at most 16
// prompt-terminated frames per call. A framing error poisons it until Reset.
// Callers preserve raw chunks before framing. CR/LF never delimit transactions.
type Framer struct {
	buffer []byte
	failed bool
}

func (f *Framer) Reset()       { f.buffer = nil; f.failed = false }
func (f *Framer) Pending() int { return len(f.buffer) }
func (f *Framer) Feed(chunk []byte) ([][]byte, error) {
	if f.failed {
		return nil, fmt.Errorf("framer poisoned")
	}
	if len(chunk) > 65536 {
		f.failed = true
		return nil, fmt.Errorf("chunk limit")
	}
	var frames [][]byte
	for _, b := range chunk {
		if b != '\r' && b != '\n' && b != '\t' && (b < 32 || b > 126) {
			f.failed = true
			return nil, fmt.Errorf("non-ASCII transport data")
		}
		if len(f.buffer) == 0 && (b == '\r' || b == '\n' || b == '\t' || b == ' ') {
			continue
		}
		f.buffer = append(f.buffer, b)
		if len(f.buffer) > 8192 {
			f.failed = true
			return nil, fmt.Errorf("response limit")
		}
		if b == '>' {
			frames = append(frames, append([]byte(nil), f.buffer...))
			f.buffer = nil
			if len(frames) > 16 {
				f.failed = true
				return nil, fmt.Errorf("frame count limit")
			}
		}
	}
	return frames, nil
}

type Failure struct{ ECU, Detail string }

type Reply struct {
	Responses []Response
	Statuses  []string
	Failures  []Failure
}
type Response struct {
	ECU     string
	Payload []byte
	State   string
	Value   *float64
}

var voltage = regexp.MustCompile(`^[0-9]{1,2}\.[0-9]{2}$`)

func compact(s string) string { return strings.ToUpper(strings.Join(strings.Fields(s), "")) }
func Parse(key string, frame []byte) (Reply, error) {
	d, err := Lookup(key)
	if err != nil {
		return Reply{}, err
	}
	if len(frame) == 0 || len(frame) > 8192 || frame[len(frame)-1] != '>' || strings.Count(string(frame), ">") != 1 {
		return Reply{}, fmt.Errorf("one complete prompt required")
	}
	for _, b := range frame {
		if b != '\r' && b != '\n' && b != '\t' && (b < 32 || b > 126) {
			return Reply{}, fmt.Errorf("invalid ASCII")
		}
	}
	lines := strings.FieldsFunc(string(frame[:len(frame)-1]), func(r rune) bool { return r == '\r' || r == '\n' })
	if len(lines) > 32 {
		return Reply{}, fmt.Errorf("line count limit")
	}
	result := Reply{}
	seen := map[string]bool{}
	bad := map[string]bool{}
	echo := false
	for _, line := range lines {
		if len(line) > 256 {
			return Reply{}, fmt.Errorf("line length limit")
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if compact(line) == compact(d.Command) {
			if echo {
				return Reply{}, fmt.Errorf("duplicate echo")
			}
			echo = true
			continue
		}
		switch line {
		case "SEARCHING...":
			result.Statuses = append(result.Statuses, line)
			continue
		case "NO DATA", "?", "STOPPED", "UNABLE TO CONNECT", "BUS ERROR", "CAN ERROR", "BUFFER FULL", "DATA ERROR", "LV RESET":
			result.Statuses = append(result.Statuses, line)
			continue
		}
		response, decodeErr := decodeLine(d, line)
		if decodeErr != nil {
			ecu := "unknown"
			text := compact(line)
			if len(text) >= 3 {
				if n, e := strconv.ParseUint(text[:3], 16, 12); e == nil && n >= 0x7e8 && n <= 0x7ef {
					ecu = text[:3]
				}
			}
			result.Failures = append(result.Failures, Failure{ecu, decodeErr.Error()})
			if ecu != "unknown" {
				bad[ecu] = true
				filtered := result.Responses[:0]
				for _, r := range result.Responses {
					if r.ECU != ecu {
						filtered = append(filtered, r)
					}
				}
				result.Responses = filtered
			}
			continue
		}
		if bad[response.ECU] {
			continue
		}
		if seen[response.ECU] {
			bad[response.ECU] = true
			filtered := result.Responses[:0]
			for _, r := range result.Responses {
				if r.ECU != response.ECU {
					filtered = append(filtered, r)
				}
			}
			result.Responses = filtered
			result.Failures = append(result.Failures, Failure{response.ECU, "duplicate responder"})
			continue
		}
		seen[response.ECU] = true
		result.Responses = append(result.Responses, response)
	}
	if len(result.Responses) == 0 && (len(result.Statuses) == 0 || len(result.Failures) > 0) {
		return Reply{}, fmt.Errorf("empty response is not a measurement")
	}
	return result, nil
}

func decodeLine(d Definition, line string) (Response, error) {
	if d.Key == "adapter_voltage" {
		if line == "--.--" {
			return Response{ECU: "adapter", State: "unavailable"}, nil
		}
		if !voltage.MatchString(line) {
			return Response{}, fmt.Errorf("unsupported voltage format")
		}
		value, _ := strconv.ParseFloat(line, 64)
		if value > 65.534 {
			return Response{}, fmt.Errorf("voltage out of documented range")
		}
		return Response{ECU: "adapter", State: "observed", Value: &value}, nil
	}
	text := compact(line)
	if len(text) < 7 || len(text)%2 != 1 {
		return Response{}, fmt.Errorf("unsupported frame format (headers/PCI required)")
	}
	id, err := strconv.ParseUint(text[:3], 16, 12)
	if err != nil || id < 0x7e8 || id > 0x7ef {
		return Response{}, fmt.Errorf("unsupported response ID")
	}
	data, err := hex.DecodeString(text[3:])
	if err != nil || len(data) > 8 || len(data) < 2 || data[0] == 0 || data[0] > 7 || int(data[0])+1 > len(data) {
		return Response{}, fmt.Errorf("unsupported/truncated ISO-TP single frame")
	}
	payload := append([]byte(nil), data[1:1+int(data[0])]...)
	response := Response{ECU: text[:3], Payload: payload, State: "advertised"}
	if len(payload) == 3 && payload[0] == 0x7f && payload[1] == 1 {
		response.State = "negative_response"
		return response, nil
	}
	if len(payload) != d.Bytes+2 || payload[0] != 0x41 || payload[1] != d.PID {
		return Response{}, fmt.Errorf("foreign service/PID or invalid payload length")
	}
	if d.Key != "support" {
		value := float64(payload[2]) - 40
		if d.Key == "rpm" {
			value = float64(binary.BigEndian.Uint16(payload[2:])) / 4
		}
		response.State = "observed"
		response.Value = &value
	}
	return response, nil
}

// Capabilities are scoped to a caller-supplied connection/configuration identity.
// Timeouts never establish unsupported status; observed is retained separately.
type Capability struct {
	ECU        string `json:"ecu"`
	Key        string `json:"key"`
	State      string `json:"state"`
	Advertised *bool  `json:"advertised"`
	Observed   bool   `json:"observed"`
	Evidence   string `json:"evidence_transaction"`
}
type Capabilities struct {
	Scope string
	rows  map[string]Capability
}

func NewCapabilities(scope string) *Capabilities {
	return &Capabilities{scope, map[string]Capability{}}
}
func (c *Capabilities) ApplySupport(response Response, tx string) error {
	if response.State != "advertised" || len(response.Payload) != 6 || response.Payload[0] != 0x41 || response.Payload[1] != 0 {
		return fmt.Errorf("valid block 00 response required")
	}
	bits := binary.BigEndian.Uint32(response.Payload[2:])
	for _, d := range catalog {
		if d.Key == "support" || d.Key == "adapter_voltage" {
			continue
		}
		yes := bits&(uint32(1)<<uint(32-d.PID)) != 0
		row := c.Get(response.ECU, d.Key)
		row.Advertised = &yes
		row.Evidence = tx
		if !row.Observed {
			if yes {
				row.State = "advertised"
			} else {
				row.State = "unsupported"
			}
		}
		c.rows[response.ECU+"/"+d.Key] = row
	}
	return nil
}
func (c *Capabilities) Get(ecu, key string) Capability {
	if row, ok := c.rows[ecu+"/"+key]; ok {
		if row.Advertised != nil {
			value := *row.Advertised
			row.Advertised = &value
		}
		return row
	}
	return Capability{ECU: ecu, Key: key, State: "unknown"}
}
func (c *Capabilities) Observe(ecu, key, tx string) {
	row := c.Get(ecu, key)
	row.State = "observed"
	row.Observed = true
	row.Evidence = tx
	c.rows[ecu+"/"+key] = row
}
func (c *Capabilities) Unavailable(ecu, key, tx string) {
	row := c.Get(ecu, key)
	if row.Observed || (row.Advertised != nil && *row.Advertised) {
		row.State = "unavailable"
	}
	row.Evidence = tx
	c.rows[ecu+"/"+key] = row
}
func (c *Capabilities) Rows() []Capability {
	var out []Capability
	// Fixed catalog and sorted caller-known ECU IDs avoid randomized map output.
	for _, ecu := range []string{"7E8", "7E9", "7EA", "7EB", "7EC", "7ED", "7EE", "7EF", "adapter"} {
		for _, d := range catalog {
			if row, ok := c.rows[ecu+"/"+d.Key]; ok {
				out = append(out, c.Get(row.ECU, row.Key))
			}
		}
	}
	return out
}
