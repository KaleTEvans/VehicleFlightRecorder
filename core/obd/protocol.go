// Package obd implements the deliberately narrow documented ASCII/CAN profile.
// It defines byte requests and decoders but cannot open or write a transport.
package obd

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
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

const ExpandedWorkload = "standard_expanded"
const DiagnosticRecordingWorkload = "standard_diagnostic_recording"
const J1979Reference = "https://www.obdlink.com/obd-apps/obdwiz-app/"

var expandedCatalog = []Definition{
	{"support20", "0120\r", "", "", J1979Reference, 0x20, 4},
	{"support40", "0140\r", "", "", J1979Reference, 0x40, 4},
	{"speed", "010D\r", "powertrain.vehicle.speed", "km/h", J1979Reference, 0x0d, 1},
	{"short_trim_bank1", "0106\r", "powertrain.fuel.trim.short.bank1", "%", J1979Reference, 0x06, 1},
	{"long_trim_bank1", "0107\r", "powertrain.fuel.trim.long.bank1", "%", J1979Reference, 0x07, 1},
	{"short_trim_bank2", "0108\r", "powertrain.fuel.trim.short.bank2", "%", J1979Reference, 0x08, 1},
	{"long_trim_bank2", "0109\r", "powertrain.fuel.trim.long.bank2", "%", J1979Reference, 0x09, 1},
	{"oil_temperature", "015C\r", "powertrain.oil.temperature", "degC", J1979Reference, 0x5c, 1},
	{"fuel_rate", "015E\r", "powertrain.fuel.rate", "L/h", J1979Reference, 0x5e, 2},
}

var diagnosticRecordingCatalog = []Definition{
	{"calculated_load", "0104\r", "powertrain.engine.calculated_load", "%", J1979Reference, 0x04, 1},
	{"manifold_pressure", "010B\r", "powertrain.intake.manifold_pressure", "kPa", J1979Reference, 0x0b, 1},
	{"intake_temperature", "010F\r", "powertrain.intake.temperature", "degC", J1979Reference, 0x0f, 1},
	{"throttle_position", "0111\r", "powertrain.throttle.position", "%", J1979Reference, 0x11, 1},
	{"fuel_rail_pressure", "0123\r", "powertrain.fuel.rail_pressure_gauge", "kPa", J1979Reference, 0x23, 2},
}
var expandedMeasurementCatalog = append(Catalog(), expandedCatalog...)
var liveMeasurementCatalog = append(append([]Definition(nil), expandedMeasurementCatalog...), diagnosticRecordingCatalog...)

func DiagnosticRecordingRequest(key string) bool {
	for _, d := range diagnosticRecordingCatalog {
		if d.Key == key {
			return true
		}
	}
	return false
}

func ExpandedRequest(key string) bool {
	for _, d := range expandedCatalog {
		if d.Key == key {
			return true
		}
	}
	return false
}

func IsSupportRequest(key string) bool {
	return key == "support" || key == "support20" || key == "support40"
}

func isFuelTrim(pid byte) bool { return pid >= 0x06 && pid <= 0x09 }

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
	return ParseProfile(Profile, key, frame)
}

// ParseProfile keeps CAN identifier/addressing rules explicit. The 29-bit
// profile accepts only normal-fixed physical replies addressed to tester F1,
// retaining the full responder ID without assigning vehicle module names.
func ParseProfile(profile, key string, frame []byte) (Reply, error) {
	d, err := lookupMeasurement(key)
	if err != nil {
		return Reply{}, err
	}
	return parseDefinition(profile, d, frame)
}

func parseDefinition(profile string, d Definition, frame []byte) (Reply, error) {
	if profile != Profile && profile != CAN29Profile {
		return Reply{}, fmt.Errorf("unsupported decoder profile")
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
		response, decodeErr := decodeLine(profile, d, line)
		if decodeErr != nil {
			ecu := "unknown"
			text := compact(line)
			if profile == CAN29Profile {
				if len(text) >= 8 && validCAN29ID(text[:8]) {
					ecu = text[:8]
				}
			} else if len(text) >= 3 {
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

func validCAN29ID(id string) bool {
	if len(id) != 8 || !strings.HasPrefix(id, "18DAF1") || id[6:] == "F1" {
		return false
	}
	_, err := strconv.ParseUint(id, 16, 29)
	return err == nil
}

func decodeLine(profile string, d Definition, line string) (Response, error) {
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
	headerLength := 3
	if profile == CAN29Profile {
		headerLength = 8
	}
	if len(text) < headerLength+4 || (len(text)-headerLength)%2 != 0 {
		return Response{}, fmt.Errorf("unsupported frame format (headers/PCI required)")
	}
	id, err := strconv.ParseUint(text[:headerLength], 16, 29)
	valid := err == nil && id >= 0x7e8 && id <= 0x7ef
	if profile == CAN29Profile {
		valid = validCAN29ID(text[:headerLength])
	}
	if !valid {
		return Response{}, fmt.Errorf("unsupported response ID")
	}
	data, err := hex.DecodeString(text[headerLength:])
	if err != nil || len(data) > 8 || len(data) < 2 || data[0] == 0 || data[0] > 7 || int(data[0])+1 > len(data) {
		return Response{}, fmt.Errorf("unsupported/truncated ISO-TP single frame")
	}
	payload := append([]byte(nil), data[1:1+int(data[0])]...)
	response := Response{ECU: text[:headerLength], Payload: payload, State: "advertised"}
	if len(payload) == 3 && payload[0] == 0x7f && payload[1] == 1 {
		response.State = "negative_response"
		return response, nil
	}
	validLength := len(payload) == d.Bytes+2 || (d.Bytes == -1 && len(payload) >= 3) || (isFuelTrim(d.PID) && len(payload) == 4)
	if !validLength || payload[0] != 0x41 || payload[1] != d.PID {
		return Response{}, fmt.Errorf("foreign service/PID or invalid payload length")
	}
	if d.Signal != "" {
		value := float64(payload[2])
		switch {
		case d.Key == "coolant" || d.Key == "oil_temperature" || d.Key == "intake_temperature":
			value -= 40
		case d.Key == "rpm":
			value = float64(binary.BigEndian.Uint16(payload[2:])) / 4
		case d.Key == "calculated_load" || d.Key == "throttle_position":
			value = value * 100 / 255
		case d.Key == "fuel_rail_pressure":
			value = float64(binary.BigEndian.Uint16(payload[2:])) * 10
		case d.Key == "fuel_rate":
			value = float64(binary.BigEndian.Uint16(payload[2:])) / 20
		case isFuelTrim(d.PID):
			// An optional second byte describes bank 3 or 4, outside this workload.
			value = (value - 128) * 100 / 128
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
	return c.applySupport(response, tx, catalog, 0)
}

func (c *Capabilities) ApplyExpandedSupport(response Response, tx string) error {
	return c.applySupport(response, tx, expandedMeasurementCatalog, 0x40)
}

func (c *Capabilities) ApplyDiagnosticRecordingSupport(response Response, tx string) error {
	return c.applySupport(response, tx, liveMeasurementCatalog, 0x20)
}

func (c *Capabilities) applySupport(response Response, tx string, definitions []Definition, maxBlock int) error {
	if response.State != "advertised" || len(response.Payload) != 6 || response.Payload[0] != 0x41 {
		return fmt.Errorf("valid support block response required")
	}
	block := int(response.Payload[1])
	if block > maxBlock || block%32 != 0 {
		return fmt.Errorf("unsupported capability block")
	}
	bits := binary.BigEndian.Uint32(response.Payload[2:])
	for _, d := range definitions {
		offset := int(d.PID) - block
		if offset <= 0 || offset > 32 || (IsSupportRequest(d.Key) && int(d.PID) > maxBlock) {
			continue
		}
		yes := bits&(uint32(1)<<uint(32-offset)) != 0
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
	// Sort actual observed addresses; never truncate 29-bit responders to 11 bits.
	ids := map[string]bool{}
	for _, row := range c.rows {
		ids[row.ECU] = true
	}
	ordered := make([]string, 0, len(ids))
	for ecu := range ids {
		ordered = append(ordered, ecu)
	}
	sort.Strings(ordered)
	for _, ecu := range ordered {
		for _, d := range liveMeasurementCatalog {
			if row, ok := c.rows[ecu+"/"+d.Key]; ok {
				out = append(out, c.Get(row.ECU, row.Key))
			}
		}
	}
	return out
}
