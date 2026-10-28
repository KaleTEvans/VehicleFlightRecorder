package mobile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	maxManualMarks          = 100
	bridgeVersion     int64 = 1
	maxJSONBytes            = 1 << 20
	maxLogBytes       int64 = 64 << 20
	maxResponseBytes        = 8 << 20
	maxHistoryEntries       = 2000
	historyPageSize         = 50
	maxTimelineRows         = 50000
)

// BridgeVersion is independent of application, telemetry and vehicle-pack versions.
func BridgeVersion() int64 { return bridgeVersion }

// CheckBridgeVersion rejects incompatible native clients before controller creation.
func CheckBridgeVersion(requested int64) error {
	if requested != bridgeVersion {
		return fault("unsupported_version", "CheckBridgeVersion", fmt.Errorf("native bridge %d, core bridge %d", requested, bridgeVersion))
	}
	return nil
}

// BridgeContractJSON describes fixed limits without initializing storage or hardware.
func BridgeContractJSON() string {
	data, _ := json.Marshal(map[string]any{"bridge_version": bridgeVersion, "snapshot_schema_version": 1,
		"max_json_bytes": maxJSONBytes, "max_response_bytes": maxResponseBytes,
		"max_log_bytes": maxLogBytes, "history_page_size": historyPageSize,
		"max_history_entries": maxHistoryEntries, "max_timeline_rows": maxTimelineRows, "max_manual_marks": maxManualMarks,
		"error_categories": []string{"invalid_input", "invalid_state", "unsupported_version", "missing_pack", "io_failure", "corrupt_data", "resource_limit", "cancelled", "transport_unavailable", "internal"}})
	return string(data)
}

type bridgeFailure struct {
	code, operation string
	cause           error
}

func (e *bridgeFailure) Error() string {
	return fmt.Sprintf("VFR|%s|%s|%s", e.code, e.operation, e.cause)
}
func (e *bridgeFailure) Unwrap() error       { return e.cause }
func fault(code, op string, err error) error { return &bridgeFailure{code, op, err} }
func wrapFailure(op string, err *error) {
	if *err == nil {
		return
	}
	var known *bridgeFailure
	if errors.As(*err, &known) {
		return
	}
	code := "internal"
	var path *os.PathError
	var link *os.LinkError
	var system *os.SyscallError
	if errors.As(*err, &path) || errors.As(*err, &link) || errors.As(*err, &system) || errors.Is(*err, io.ErrShortWrite) {
		code = "io_failure"
	}
	*err = fault(code, op, *err)
}
func (c *Controller) checkOpen() error {
	if c.closed {
		return fault("invalid_state", "Controller", fmt.Errorf("controller is closed; create a new instance"))
	}
	return nil
}
