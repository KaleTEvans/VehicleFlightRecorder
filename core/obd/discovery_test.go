package obd

import "testing"

func TestDiscoveryFollowsAdvertisedBlocksAndProbesOnce(t *testing.T) {
	pass, err := NewDiscoveryPass(CAN29Profile)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		pid     byte
		block   bool
		frame   string
		outcome string
	}{
		{0x00, true, "18DAF110 06 41 00 00 18 00 01\r18DAF117 06 41 00 00 10 00 00\r>", "reply"},
		{0x20, true, "18DAF110 06 41 20 80 00 00 01\r>", "reply"},
		{0x40, true, "18DAF110 06 41 40 00 00 00 04\r>", "reply"},
		{0x0c, false, "18DAF110 04 41 0C 0F A0\r>", "reply"},
		{0x0d, false, "NO DATA\r>", "no_data"},
		{0x21, false, "", "timeout"},
		{0x5e, false, "18DAF110 04 41 0D 00 01\r>", "unparsed"},
	} {
		request, ok := pass.Next()
		if !ok || request.PID != step.pid || request.SupportBlock != step.block {
			t.Fatalf("unexpected request: %+v", request)
		}
		if _, ok := pass.Next(); ok {
			t.Fatal("admitted a second outstanding request")
		}
		result, err := pass.Complete(request, []byte(step.frame))
		if err != nil || result.Outcome != step.outcome {
			t.Fatalf("PID %02X: %+v, %v", request.PID, result, err)
		}
		if step.pid == 0x0c && (result.Responders[0].Value != nil || result.Responders[0].State != "positive_reply") {
			t.Fatal("raw probe was treated as a decoded reading")
		}
	}
	if request, ok := pass.Next(); ok {
		t.Fatalf("unexpected work after bounded pass: %+v", request)
	}
}
