package obd

import (
	"encoding/binary"
	"fmt"
	"sort"
)

const DiscoveryWorkload = "standard_pid_discovery"

// DiscoveryPass plans one read-only Mode 01 pass for one connection. The
// transport owns write admission, timeouts, raw evidence and synchronization.
// Discard the pass on reconnect so prior capability bits cannot admit requests.
type DiscoveryPass struct {
	profile    string
	blocks     []byte
	candidates map[byte]bool
	probes     []byte
	current    *DiscoveryRequest
}

// DiscoveryRequest identifies the single outstanding request. Complete it with
// one framed reply or a timeout before requesting further work.
type DiscoveryRequest struct {
	PID          byte
	SupportBlock bool
	Command      string
}

type DiscoveryResult struct {
	Request    DiscoveryRequest
	Outcome    string
	Responders []Response
}

func NewDiscoveryPass(profile string) (*DiscoveryPass, error) {
	if profile != Profile && profile != CAN29Profile {
		return nil, fmt.Errorf("unsupported discovery profile")
	}
	return &DiscoveryPass{profile: profile, blocks: []byte{0}, candidates: map[byte]bool{}}, nil
}

func (p *DiscoveryPass) Next() (DiscoveryRequest, bool) {
	if p.current != nil {
		return DiscoveryRequest{}, false
	}
	var request DiscoveryRequest
	if len(p.blocks) > 0 {
		request.PID, p.blocks = p.blocks[0], p.blocks[1:]
		request.SupportBlock = true
	} else {
		if p.probes == nil {
			p.probes = make([]byte, 0, len(p.candidates))
			for pid := range p.candidates {
				p.probes = append(p.probes, pid)
			}
			sort.Slice(p.probes, func(i, j int) bool { return p.probes[i] < p.probes[j] })
		}
		if len(p.probes) == 0 {
			return DiscoveryRequest{}, false
		}
		request.PID, p.probes = p.probes[0], p.probes[1:]
	}
	request.Command = fmt.Sprintf("01%02X\r", request.PID)
	p.current = &request
	return request, true
}

// Complete consumes an explicit terminal outcome. Only validated single-frame
// bitmap replies advance discovery. Unknown data PIDs retain bytes and source
// without assigning units or claiming a decoded measurement.
func (p *DiscoveryPass) Complete(request DiscoveryRequest, frame []byte) (DiscoveryResult, error) {
	if p.current == nil || request != *p.current {
		return DiscoveryResult{}, fmt.Errorf("reply does not match outstanding discovery request")
	}
	p.current = nil
	result := DiscoveryResult{Request: request, Outcome: "timeout"}
	if len(frame) == 0 {
		return result, nil
	}
	def := Definition{PID: request.PID, Command: request.Command, Bytes: -1}
	if request.SupportBlock {
		def.Bytes = 4
	}
	reply, err := parseDefinition(p.profile, def, frame)
	result.Outcome = "unparsed"
	if err != nil {
		return result, nil
	}
	result.Responders = reply.Responses
	if len(reply.Responses) == 0 && len(reply.Failures) == 0 &&
		len(reply.Statuses) > 0 && reply.Statuses[len(reply.Statuses)-1] == "NO DATA" {
		result.Outcome = "no_data"
		return result, nil
	}
	continuation := false
	for i, response := range reply.Responses {
		if response.State != "advertised" {
			if result.Outcome == "unparsed" && response.State == "negative_response" {
				result.Outcome = "negative_response"
			}
			continue
		}
		result.Outcome = "reply"
		if !request.SupportBlock {
			result.Responders[i].State = "positive_reply"
			continue
		}
		mask := binary.BigEndian.Uint32(response.Payload[2:])
		for offset := byte(1); offset < 32; offset++ {
			if mask&(uint32(1)<<uint(32-offset)) != 0 {
				p.candidates[request.PID+offset] = true
			}
		}
		continuation = continuation || mask&1 != 0
	}
	if request.SupportBlock && continuation && request.PID < 0xE0 {
		p.blocks = append(p.blocks, request.PID+0x20)
	}
	return result, nil
}
