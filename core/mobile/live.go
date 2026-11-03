package mobile

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"vehicleflightrecorder.local/core/acquisition"
	"vehicleflightrecorder.local/core/incidents"
	"vehicleflightrecorder.local/core/obd"
	"vehicleflightrecorder.local/core/recorder"
	"vehicleflightrecorder.local/core/telemetry"
)

const liveLimitNS int64 = 600 * acquisition.Second

type liveOptions struct {
	Origin         string `json:"origin"`
	OperatingState string `json:"operating_state"`
	Workload       string `json:"workload"`
	ReferenceNote  string `json:"reference_note"`
}

// LiveRecorder serializes Go admission/storage. Its caller supplies native
// continuous-clock nanoseconds; no UI refresh, Go ticker or mock clock polls a car.
// No public API accepts an arbitrary command string or manufacturer pack request.
type LiveRecorder struct {
	mu                                                       sync.Mutex
	core                                                     *recorder.Recorder
	scheduler                                                *acquisition.Scheduler
	options                                                  liveOptions
	path                                                     string
	baseNS, anchorMS, elapsed, lastOffset, lastTick          int64
	pending                                                  []byte
	observationCursor, marks, resyncs                        int
	connected, ready, connectedOnce, reconnectNeeded, failed bool
	failure                                                  string
	cached                                                   map[string]acquisition.Observation
}

func NewLiveRecorder(root, options string, wallMS, nativeNS int64) (result *LiveRecorder, err error) {
	defer wrapFailure("NewLiveRecorder", &err)
	var config liveOptions
	if !filepath.IsAbs(root) || wallMS <= 0 || nativeNS < 0 {
		return nil, fault("invalid_input", "NewLiveRecorder", fmt.Errorf("absolute root and valid clocks required"))
	}
	if err = decodeDocument([]byte(options), &config); err != nil {
		return nil, err
	}
	if len(config.ReferenceNote) > 512 {
		return nil, fmt.Errorf("reference note exceeds 512 bytes")
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	id := "live-" + hex.EncodeToString(nonce[:])
	metadata := telemetry.Session{SchemaVersion: 2, SessionID: id, Source: "obd-live", ProfileID: "generic-obd-can11-500", VehiclePackVersion: "none", StartTimeUnixMS: wallMS, State: "recording", Live: &telemetry.LiveMetadata{Origin: config.Origin, OperatingState: config.OperatingState, Workload: config.Workload, NativeAnchorNS: nativeNS, Catalog: obd.LiveCatalogVersion, Protocol: obd.Profile, ReportFile: "capability.json", StartOrigin: "manual"}}
	if err = metadata.Validate(); err != nil {
		return nil, err
	}
	specs := []acquisition.Schedule{{Key: "support", IntervalNS: 60 * acquisition.Second, TimeoutNS: 2 * acquisition.Second, Priority: 0}}
	for _, spec := range []acquisition.Schedule{{Key: "rpm", IntervalNS: 2 * acquisition.Second, TimeoutNS: 2 * acquisition.Second, Priority: 1}, {Key: "coolant", IntervalNS: 5 * acquisition.Second, TimeoutNS: 2 * acquisition.Second, Priority: 2}, {Key: "adapter_voltage", IntervalNS: 5 * acquisition.Second, TimeoutNS: 2 * acquisition.Second, Priority: 3}} {
		if config.Workload == "combined" || (config.Workload == "rpm_only" && spec.Key == "rpm") || (config.Workload == "coolant_only" && spec.Key == "coolant") || (config.Workload == "voltage_only" && spec.Key == "adapter_voltage") {
			specs = append(specs, spec)
		}
	}
	// Voltage-only testing does not query an ECU merely to measure the adapter.
	if config.Workload == "voltage_only" {
		specs = specs[1:]
	}
	engine, err := acquisition.NewLive(wallMS, specs, acquisition.LiveConfiguration{SessionID: id, Origin: config.Origin, OperatingState: config.OperatingState, Workload: config.Workload, NativeAnchorNS: nativeNS, Initialization: obd.InitializationVersion, Protocol: "ISO15765-4-CAN11-500", AppVersion: "0.1.0", BridgeVersion: BridgeVersion(), ReferenceNote: config.ReferenceNote, SensorRefresh: "unknown", EnhancedPack: "none", MaximumDurationNS: liveLimitNS, MinimumAdmissionNS: acquisition.Second / 2, Connections: []acquisition.Connection{}})
	if err != nil {
		return nil, err
	}
	core, err := recorder.New(60 * time.Second)
	if err != nil {
		return nil, err
	}
	if err = core.ConfigureIncidents(incidents.Config{Availability: "manual_only", Provenance: "manual-physical-capture-1", PreWindowMS: 60000, PostWindowMS: 120000, MaxSilenceMS: 15000}); err != nil {
		return nil, err
	}
	path := filepath.Join(root, id)
	if err = core.Start(path, metadata); err != nil {
		return nil, err
	}
	l := &LiveRecorder{core: core, scheduler: engine, options: config, path: path, baseNS: nativeNS, anchorMS: wallMS, cached: map[string]acquisition.Observation{}}
	engine.SetRawSink(core.AppendTransport)
	if err = l.checkpoint(); err != nil {
		core.Close()
		return nil, err
	}
	return l, nil
}

func (l *LiveRecorder) clock(nativeNS int64) (int64, error) {
	if nativeNS < l.baseNS || nativeNS-l.baseNS < l.elapsed {
		return 0, fmt.Errorf("native clock regression")
	}
	ns := nativeNS - l.baseNS
	if ns > 24*60*60*acquisition.Second {
		return 0, fmt.Errorf("live clock exceeds 24-hour bound")
	}
	l.elapsed = ns
	return ns, nil
}
func (l *LiveRecorder) event(kind, severity string, attributes any) error {
	data, err := json.Marshal(attributes)
	if err != nil {
		return err
	}
	return l.core.Append(telemetry.Record{Event: &telemetry.Event{SchemaVersion: 2, SessionID: l.core.Session().SessionID, Source: "obd-live", TimestampUnixMS: l.anchorMS + l.elapsed/1_000_000, EventType: kind, Severity: severity, Attributes: data}})
}
func (l *LiveRecorder) checkpoint() error {
	r := l.scheduler.Report()
	// The selected protocol is independently validated by init_protocol_check;
	// observed ECU support/values are separate from adapter selection.
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(b) > 16<<20 {
		return fmt.Errorf("live capability report exceeds 16 MiB")
	}
	file, err := os.CreateTemp(l.path, ".capability-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if err = file.Chmod(0600); err == nil {
		data := append(b, '\n')
		var n int
		n, err = file.Write(data)
		if err == nil && n != len(data) {
			err = io.ErrShortWrite
		}
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(l.path, "capability.json"))
}
func (l *LiveRecorder) fail(err error) error {
	l.failed = true
	l.failure = err.Error()
	l.reconnectNeeded = false
	l.core.Close()
	return err
}
func (l *LiveRecorder) consume() error {
	for _, o := range l.scheduler.ObservationsSince(l.observationCursor) {
		tx, ok := l.scheduler.TransactionFor(o.Transaction)
		if !ok || tx.SentNS == nil {
			return fmt.Errorf("observation transaction missing")
		}
		value := telemetry.Observation{SchemaVersion: 2, TimestampUnixMS: o.MappedMS, SessionID: l.core.Session().SessionID, SignalID: o.Signal, Source: "obd-live", Value: o.Value, Unit: o.Unit, Quality: "valid", Module: o.ECU, Bus: "HS-CAN", DecoderVersion: o.Decoder, VehiclePackVersion: "none", Provenance: &telemetry.Provenance{Transaction: o.Transaction, Generation: o.Generation, RawRefs: append([]int(nil), tx.RawRefs...), StartNS: tx.StartNS, SentNS: *tx.SentNS, ReceiveNS: o.ReceiveNS}}
		if err := l.core.Append(telemetry.Record{Observation: &value}); err != nil {
			return err
		}
		l.cached[o.Signal+"/"+o.ECU] = o
		l.observationCursor++
	}
	return nil
}
func (l *LiveRecorder) Connected(metadata string, nativeNS int64) (generation int64, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer wrapFailure("LiveConnected", &err)
	if l.failed || l.core.State() != "recording" || l.connected {
		return 0, fmt.Errorf("session cannot accept a connection")
	}
	now, err := l.clock(nativeNS)
	if err != nil {
		return 0, err
	}
	var attrs map[string]string
	if err = decodeDocument([]byte(metadata), &attrs); err != nil {
		return 0, err
	}
	if len(attrs) > 16 || attrs["protocol"] != "com.obdlink" {
		return 0, fmt.Errorf("invalid live adapter metadata")
	}
	for k, v := range attrs {
		if len(k) > 64 || len(v) > 512 {
			return 0, fmt.Errorf("adapter metadata budget")
		}
	}
	if l.scheduler.Generation() >= 16 {
		return 0, l.fail(fmt.Errorf("session generation budget exhausted"))
	}
	if l.connectedOnce {
		if err = l.scheduler.Reconnect(now); err != nil {
			return 0, err
		}
	}
	l.connected = true
	l.ready = false
	l.connectedOnce = true
	l.reconnectNeeded = false
	l.failure = ""
	l.pending = nil
	l.cached = map[string]acquisition.Observation{}
	if err = l.scheduler.AddConnection(now, attrs); err != nil {
		return 0, err
	}
	if err = l.checkpoint(); err != nil {
		return 0, l.fail(err)
	}
	if err = l.event("adapter_session_created", "info", map[string]any{"generation": l.scheduler.Generation(), "streams": "opening"}); err != nil {
		return 0, l.fail(err)
	}
	return int64(l.scheduler.Generation()), nil
}
func (l *LiveRecorder) Disconnected(generation, nativeNS int64, reason string) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer wrapFailure("LiveDisconnected", &err)
	if l.core.State() != "recording" || !l.connected {
		return nil
	}
	if generation != int64(l.scheduler.Generation()) {
		return fmt.Errorf("stale live generation")
	}
	now, err := l.clock(nativeNS)
	if err != nil {
		return err
	}
	l.connected = false
	l.ready = false
	l.pending = nil
	if err = l.scheduler.Stop(now); err != nil {
		return l.fail(err)
	}
	if err = l.checkpoint(); err != nil {
		return l.fail(err)
	}
	return l.event("transport_disconnected", "warning", map[string]any{"generation": generation, "reason": reason})
}
func (l *LiveRecorder) NextCommand(nativeNS, wallMS int64) (data []byte, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer wrapFailure("LiveNextCommand", &err)
	if l.failed {
		return nil, fmt.Errorf("live session failed: %s", l.failure)
	}
	if l.reconnectNeeded {
		return nil, fmt.Errorf("%s", l.failure)
	}
	if l.core.State() != "recording" {
		return nil, nil
	}
	now, err := l.clock(nativeNS)
	if err != nil {
		return nil, l.fail(err)
	}
	if wallMS > 0 {
		offset := wallMS - (l.anchorMS + now/1_000_000)
		if delta := offset - l.lastOffset; delta > 1000 || delta < -1000 {
			l.lastOffset = offset
			if err = l.event("wall_clock_adjusted", "warning", map[string]any{"observed_unix_ms": wallMS, "offset_ms": offset, "canonical_mapping": "unchanged"}); err != nil {
				return nil, l.fail(err)
			}
		}
	}
	if l.lastTick > 0 && now-l.lastTick > 3*acquisition.Second {
		if err = l.event("acquisition_pause_observed", "warning", map[string]any{"from_ns": l.lastTick, "to_ns": now, "cause": "unknown_native_scheduling_gap"}); err != nil {
			return nil, l.fail(err)
		}
	}
	l.lastTick = now
	if now >= liveLimitNS {
		if err = l.stop("stationary_test_duration_limit"); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if !l.connected || !l.ready {
		return nil, nil
	}
	work, pollErr := l.scheduler.Poll(now)
	if l.scheduler.State() == "blocked" {
		l.pending = nil
		l.resyncs++
		report := l.scheduler.Report()
		resource := false
		for _, issue := range report.Issues {
			if issue == "raw_budget_exhausted" || issue == "raw_preservation_failed" || issue == "transaction_budget_exhausted" || issue == "observation_budget_exhausted" {
				resource = true
			}
		}
		if err = l.checkpoint(); err != nil {
			return nil, l.fail(err)
		}
		if resource || l.resyncs > 3 {
			return nil, l.fail(fmt.Errorf("live acquisition stopped: evidence or resynchronization budget exhausted"))
		}
		l.reconnectNeeded = true
		l.failure = "Fresh connection required after acquisition synchronization failure"
		return nil, fmt.Errorf("%s", l.failure)
	}
	if pollErr != nil {
		return nil, l.fail(pollErr)
	}
	if work == nil {
		return nil, nil
	}
	l.pending = append([]byte(nil), work.Bytes...)
	if err = l.checkpoint(); err != nil {
		return nil, l.fail(err)
	}
	return append([]byte(nil), work.Bytes...), nil
}

// NextCommandBase64 avoids gomobile's nullable NSData/NSError import: Swift
// treats nil bytes as a throwing-method failure even when Go means no work.
// Empty string means idle; nonempty text encodes the exact Go-owned command.
func (l *LiveRecorder) NextCommandBase64(nativeNS, wallMS int64) (string, error) {
	data, err := l.NextCommand(nativeNS, wallMS)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
func (l *LiveRecorder) Accepted(generation, nativeNS int64, data []byte) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer wrapFailure("LiveAccepted", &err)
	if !l.connected || l.failed || generation != int64(l.scheduler.Generation()) {
		return fmt.Errorf("inactive/stale live write")
	}
	now, err := l.clock(nativeNS)
	if err != nil {
		return err
	}
	if len(data) == 0 || !bytes.HasPrefix(l.pending, data) {
		return fmt.Errorf("accepted bytes differ from Go request")
	}
	// Resolve the sole active transaction from its immutable request identity.
	report := l.scheduler.Report()
	if len(report.Transactions) == 0 {
		return fmt.Errorf("no active transaction")
	}
	tx := report.Transactions[len(report.Transactions)-1]
	err = l.scheduler.Accepted(int(generation), tx.ID, now, len(data))
	l.pending = l.pending[len(data):]
	if e := l.checkpoint(); e != nil {
		return l.fail(e)
	}
	return err
}
func (l *LiveRecorder) Receive(generation, nativeNS int64, data []byte) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer wrapFailure("LiveReceive", &err)
	if !l.connected || l.failed || generation != int64(l.scheduler.Generation()) {
		return fmt.Errorf("inactive/stale live receive")
	}
	now, err := l.clock(nativeNS)
	if err != nil {
		return err
	}
	parseErr := l.scheduler.Receive(int(generation), now, data)
	if err = l.checkpoint(); err != nil {
		return l.fail(err)
	}
	if err = l.consume(); err != nil {
		return l.fail(err)
	}
	if parseErr != nil && l.scheduler.State() != "blocked" {
		if err = l.event("acquisition_response_error", "warning", map[string]any{"generation": generation, "detail": parseErr.Error()}); err != nil {
			return l.fail(err)
		}
	}
	// Complete malformed replies retain failed outcomes and back off; unrelated
	// requests continue. Synchronization faults are handled by the owner tick.
	return nil
}

// NativeEvent records stream readiness/errors without admitting an adapter command.
func (l *LiveRecorder) NativeEvent(generation, nativeNS int64, kind, detail string) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer wrapFailure("LiveNativeEvent", &err)
	if l.core.State() != "recording" || l.failed {
		return nil
	}
	if !l.connected || generation != int64(l.scheduler.Generation()) {
		return fmt.Errorf("stale native event")
	}
	if len(detail) > 512 {
		return fmt.Errorf("native detail exceeds budget")
	}
	if _, err = l.clock(nativeNS); err != nil {
		return err
	}
	switch kind {
	case "streams_ready":
		if l.ready {
			return nil
		}
		l.ready = true
		return l.event("transport_connected", "info", map[string]any{"generation": generation, "scope": "adapter_streams_only", "initialization": "pending"})
	case "error":
		return l.event("native_transport_error", "warning", map[string]any{"generation": generation, "detail": detail})
	default:
		return fmt.Errorf("unknown native event")
	}
}
func (l *LiveRecorder) Lifecycle(phase string, nativeNS int64) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer wrapFailure("LiveLifecycle", &err)
	if l.core.State() != "recording" {
		return nil
	}
	if phase != "foreground" && phase != "background" && phase != "inactive" {
		return fmt.Errorf("unknown phase")
	}
	if _, err = l.clock(nativeNS); err != nil {
		return err
	}
	return l.event("native_"+phase, "info", map[string]any{"execution_continuity": "unverified"})
}
func (l *LiveRecorder) Mark(label string, nativeNS int64) (id string, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer wrapFailure("LiveMark", &err)
	if l.failed || l.core.State() != "recording" || l.marks >= maxManualMarks {
		return "", fmt.Errorf("manual marking unavailable")
	}
	if _, err = l.clock(nativeNS); err != nil {
		return "", err
	}
	id, err = l.core.MarkIncident(label, l.anchorMS+l.elapsed/1_000_000)
	if err == nil {
		l.marks++
	}
	return
}
func (l *LiveRecorder) stop(reason string) error {
	if err := l.scheduler.Stop(l.elapsed); err != nil {
		return l.fail(err)
	}
	l.pending = nil
	l.connected = false
	l.ready = false
	if err := l.checkpoint(); err != nil {
		return l.fail(err)
	}
	if err := l.event("stationary_acquisition_ended", "info", map[string]any{"reason": reason}); err != nil {
		return l.fail(err)
	}
	if err := l.core.Stop(l.anchorMS + l.elapsed/1_000_000); err != nil {
		return l.fail(err)
	}
	return nil
}
func (l *LiveRecorder) Stop(nativeNS int64) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer wrapFailure("LiveStop", &err)
	if l.core.State() == "complete" {
		return nil
	}
	if l.failed {
		return fmt.Errorf("failed session remains incomplete; preserve Files evidence")
	}
	if _, err = l.clock(nativeNS); err != nil {
		return err
	}
	return l.stop("manual")
}
func (l *LiveRecorder) Path() string { return l.path }
func (l *LiveRecorder) RequiresReconnect() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reconnectNeeded && !l.failed
}
func (l *LiveRecorder) Snapshot() (result string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	state := l.scheduler.State()
	if !l.connected {
		state = "waiting_for_adapter"
	}
	if l.connected && !l.scheduler.Initialized() {
		state = "initializing"
	}
	if l.failed {
		state = "failed"
	}
	if l.core.State() == "complete" {
		state = "complete"
	}
	readings := []map[string]any{}
	keys := []string{}
	for key := range l.cached {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		o := l.cached[key]
		quality := "valid"
		if !l.connected && l.core.State() == "recording" {
			quality = "unavailable"
		}
		for _, row := range l.scheduler.CapabilityRows() {
			d, e := obd.Lookup(row.Key)
			if e == nil && d.Signal == o.Signal && row.ECU == o.ECU && row.State == "unavailable" {
				quality = "unavailable"
			}
		}
		if quality == "valid" && l.elapsed-o.ReceiveNS > 15*acquisition.Second {
			quality = "stale"
		}
		readings = append(readings, map[string]any{"signal_id": o.Signal, "ecu": o.ECU, "value": o.Value, "unit": o.Unit, "quality": quality, "age_ms": (l.elapsed - o.ReceiveNS) / 1_000_000, "generation": o.Generation})
	}
	capabilities := l.scheduler.CapabilityRows()
	b, _ := json.Marshal(map[string]any{"state": state, "session": l.core.Session(), "elapsed_ms": l.elapsed / 1_000_000, "maximum_duration_ms": liveLimitNS / 1_000_000, "generation": l.scheduler.Generation(), "readings": readings, "capabilities": capabilities, "initialized": l.scheduler.Initialized(), "incidents": l.core.Incidents(), "error": l.failure, "requires_reconnect": l.reconnectNeeded, "start_origin": "manual", "source": "obd-live"})
	return string(b)
}
