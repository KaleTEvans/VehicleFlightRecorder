package obd

import (
	"reflect"
	"testing"
)

func TestFaultReplyDistinguishesEmptyCodesAndMissingData(t *testing.T) {
	for _, tc := range []struct {
		frame, outcome string
		codes          []string
	}{
		{"03\r7E8 02 43 00\r>", "ok", []string{}},
		{"7E8 04 43 01 01 33\r>", "ok", []string{"P0133"}},
		{"NO DATA\r>", "no_data", nil},
		{"7E8 03 7F 03 11\r>", "negative_response", nil},
		{"7E8 04 43 02 01 33\r>", "unavailable", nil},
	} {
		reply, err := ParseFaultReply(Profile, ConfirmedCodes, []byte(tc.frame))
		if err != nil || reply.Outcome != tc.outcome {
			t.Fatalf("%q: %+v, %v", tc.frame, reply, err)
		}
		if tc.codes != nil && (len(reply.Responders) != 1 || !reflect.DeepEqual(reply.Responders[0].Codes, tc.codes)) {
			t.Fatalf("%q: wrong DTCs: %+v", tc.frame, reply)
		}
		if tc.codes == nil && tc.outcome != "negative_response" && len(reply.Responders) != 0 {
			t.Fatalf("%q: invented a code response", tc.frame)
		}
	}
}

func TestFaultReplyReassemblesInterleavedECUsAndRejectsBadSequence(t *testing.T) {
	first := "18DAF110 10 08 47 03 01 33 03 00\r"
	other := "18DAF117 02 47 00\r"
	for _, tc := range []struct {
		tail, outcome string
	}{
		{"18DAF110 21 43 00 00 00 00 00 00\r>", "ok"},
		{"18DAF110 22 43 00 00 00 00 00 00\r>", "partial"},
		{">", "partial"},
	} {
		reply, err := ParseFaultReply(CAN29Profile, PendingCodes, []byte(first+other+tc.tail))
		if err != nil || reply.Outcome != tc.outcome {
			t.Fatalf("%+v %v", reply, err)
		}
		if tc.outcome == "ok" {
			if len(reply.Responders) != 2 || !reflect.DeepEqual(reply.Responders[0].Codes, []string{"P0133", "P0300", "C0300"}) {
				t.Fatalf("incorrect reassembly: %+v", reply)
			}
		} else if len(reply.Responders) != 1 || reply.Responders[0].ECU != "18DAF117" || reply.Responders[0].Outcome != "no_codes" {
			t.Fatalf("bad ECU contaminated valid evidence: %+v", reply)
		}
	}
}

func TestFaultMonitorAndFreezeFrameRequireExactReplyIdentity(t *testing.T) {
	status, err := ParseFaultReply(Profile, MonitorStatus, []byte("7E8 06 41 01 81 07 02 01\r>"))
	if err != nil || status.Outcome != "ok" || len(status.Responders) != 1 ||
		!*status.Responders[0].MIL || *status.Responders[0].CodeCount != 1 ||
		!reflect.DeepEqual(status.Responders[0].Data, []byte{0x81, 7, 2, 1}) {
		t.Fatalf("incorrect status: %+v %v", status, err)
	}
	for _, tc := range []struct {
		query FaultQuery
		line  string
		want  string
	}{
		{PermanentCodes, "7E8 04 4A 01 03 00", "ok"},
		{PermanentCodes, "7E8 04 47 01 03 00", "unavailable"},
		{FreezeFrameDTC, "7E8 05 42 02 00 03 00", "ok"},
		{FreezeFrameDTC, "7E8 05 42 02 00 00 00", "ok"},
		{FreezeFrameRPM, "7E8 05 42 0C 01 0F A0", "unavailable"},
		{FreezeFrameRPM, "7E8 05 42 0C 00 0F A0", "ok"},
		{FreezeFrameECT, "7E8 04 42 05 00 7B", "ok"},
	} {
		got, err := ParseFaultReply(Profile, tc.query, []byte(tc.line+"\r>"))
		if err != nil || got.Outcome != tc.want {
			t.Fatalf("%q: %+v %v", tc.line, got, err)
		}
	}
	if _, ok := FaultQuery("04").Command(); ok {
		t.Fatal("fault code clearing admitted")
	}
}
