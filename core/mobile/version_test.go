package mobile_test

import (
	"testing"

	"vehicleflightrecorder.local/core/mobile"
)

func TestVersion(t *testing.T) {
	if got := mobile.Version(); got != "0.1.0" {
		t.Fatalf("Version() = %q, want 0.1.0", got)
	}
}
