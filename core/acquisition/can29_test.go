package acquisition

import (
	"testing"
	"vehicleflightrecorder.local/core/obd"
)

func TestCAN29ConfigurationCannotCrossProfiles(t *testing.T) {
	base := LiveConfiguration{SessionID: "live-test", Origin: "scripted-fixture", Workload: obd.CAN29Workload, Initialization: obd.CAN29InitializationVersion, Protocol: "ISO15765-4-CAN29-500"}
	specs := []Schedule{{Key: "support", IntervalNS: 15 * Second, TimeoutNS: 2 * Second}}
	if _, err := NewLive(1700000000000, specs, base); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*LiveConfiguration){
		func(c *LiveConfiguration) { c.Protocol = "ISO15765-4-CAN11-500" },
		func(c *LiveConfiguration) { c.Initialization = obd.DiagnosticInitializationVersion },
		func(c *LiveConfiguration) { c.Workload = "combined" },
		func(c *LiveConfiguration) { c.Workload = obd.DiagnosticWorkload },
		func(c *LiveConfiguration) { c.Workload = obd.HandoffWorkload },
	} {
		config := base
		change(&config)
		if _, err := NewLive(1700000000000, specs, config); err == nil {
			t.Fatalf("accepted crossed configuration %+v", config)
		}
	}
}
