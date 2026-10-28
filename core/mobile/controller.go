package mobile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"vehicleflightrecorder.local/core/incidents"
	"vehicleflightrecorder.local/core/mock"
	"vehicleflightrecorder.local/core/recorder"
	"vehicleflightrecorder.local/core/telemetry"
	"vehicleflightrecorder.local/core/vehicle"
)

type generated struct {
	record telemetry.Record
	done   bool
	err    error
}

// Controller serializes every recorder operation. Its only producer goroutine
// owns immutable generator inputs and a bounded channel; it never touches core.
type Controller struct {
	mu           sync.Mutex
	pack         string
	root         string
	profile      vehicle.Profile
	rules        incidents.Config
	core         *recorder.Recorder
	config       mock.Config
	clock        int64
	signals      map[string]telemetry.Observation
	modules      map[string]string
	timeouts     map[string]int
	disconnected bool
	source       chan generated
	producerDone chan struct{}
	cancel       context.CancelFunc
	pending      *generated
	lastError    string
	markCount    int
	closed       bool
	failed       bool
	closeErr     error
}

func NewController(packDirectory, sessionsDirectory string) (result *Controller, err error) {
	defer wrapFailure("NewController", &err)
	if !filepath.IsAbs(packDirectory) || !filepath.IsAbs(sessionsDirectory) {
		return nil, fault("invalid_input", "NewController", fmt.Errorf("pack and sessions directories must be absolute"))
	}
	for _, name := range []string{"manifest.json", "signals.json", "thresholds.json"} {
		info, checkErr := os.Stat(filepath.Join(packDirectory, name))
		if errors.Is(checkErr, os.ErrNotExist) {
			return nil, fault("missing_pack", "NewController", checkErr)
		}
		if checkErr != nil {
			return nil, checkErr
		}
		if !info.Mode().IsRegular() || info.Size() > maxJSONBytes {
			return nil, fault("resource_limit", "NewController", fmt.Errorf("pack %s exceeds document limit or is not regular", name))
		}
	}
	p, err := vehicle.Load(packDirectory)
	if err != nil {
		return nil, fault("corrupt_data", "NewController", err)
	}
	rules, err := vehicle.LoadIncidentConfig(packDirectory, p)
	if err != nil {
		return nil, fault("corrupt_data", "NewController", err)
	}
	if err := os.MkdirAll(sessionsDirectory, 0700); err != nil {
		return nil, err
	}
	packDirectory, err = filepath.EvalSymlinks(packDirectory)
	if err != nil {
		return nil, err
	}
	sessionsDirectory, err = filepath.EvalSymlinks(sessionsDirectory)
	if err != nil {
		return nil, err
	}
	return &Controller{pack: packDirectory, root: sessionsDirectory, profile: p, rules: rules, signals: map[string]telemetry.Observation{}, modules: map[string]string{}}, nil
}

func (c *Controller) Start(scenarioName string) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("Start", &err)
	if err := c.checkOpen(); err != nil {
		return err
	}
	if c.core != nil && c.core.State() == "recording" {
		return fault("invalid_state", "Controller", fmt.Errorf("a drive is already recording"))
	}
	if c.failed || (c.core != nil && c.core.State() == "failed") {
		return fault("invalid_state", "Controller", fmt.Errorf("recorder failed; restart the controller before starting another drive"))
	}
	if !telemetry.ValidScenario(scenarioName) {
		return fault("invalid_input", "Start", fmt.Errorf("unknown mock scenario"))
	}
	var scenario *vehicle.Scenario
	if scenarioName != "healthy_drive" {
		file, checkErr := openEvidence(c.pack, "scenarios/"+scenarioName+".json", maxJSONBytes)
		if checkErr != nil {
			return checkErr
		}
		file.Close()
		value, err := vehicle.LoadScenario(c.pack, scenarioName, c.profile)
		if err != nil {
			return fault("corrupt_data", "Start", err)
		}
		scenario = &value
	}
	now := time.Now()
	config := mock.Config{SessionID: fmt.Sprintf("mock-%d", now.UnixNano()), StartTimeUnixMS: now.UnixMilli(), DurationMS: 720000, Seed: 42, Scenario: scenarioName}
	metadata, err := config.Session(c.profile)
	if err != nil {
		return err
	}
	core, err := recorder.New(time.Duration(max(c.rules.PreWindowMS, 1)) * time.Millisecond)
	if err != nil {
		return err
	}
	if err := core.ConfigureIncidents(c.rules); err != nil {
		return err
	}
	if err := core.Start(filepath.Join(c.root, metadata.SessionID), metadata); err != nil {
		return err
	}
	c.core = core
	c.config = config
	c.clock = config.StartTimeUnixMS
	c.lastError = ""
	c.markCount = 0
	c.signals = map[string]telemetry.Observation{}
	c.modules = map[string]string{}
	c.timeouts = map[string]int{}
	c.disconnected = false
	c.pending = nil
	for _, module := range c.rules.CriticalModules {
		c.modules[module] = "not_observed"
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.source = make(chan generated, 64)
	c.producerDone = make(chan struct{})
	source, done, profile := c.source, c.producerDone, c.profile
	go func() {
		defer close(done)
		defer close(source)
		err := mock.Run(ctx, config, profile, scenario, func(record telemetry.Record) error {
			select {
			case source <- generated{record: record}:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		select {
		case source <- generated{done: true, err: err}:
		case <-ctx.Done():
		}
	}()
	return c.advance(0)
}

// Advance admits at most 10 logical seconds per call. Swift schedules one call
// at a time; suspension pauses logical time rather than inventing background data.
func (c *Controller) Advance(milliseconds int64) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("Advance", &err)
	if err := c.checkOpen(); err != nil {
		return err
	}
	if milliseconds < 100 || milliseconds > 10000 || milliseconds%100 != 0 {
		return fault("invalid_input", "Advance", fmt.Errorf("advance requires 100–10000 ms in 100 ms steps"))
	}
	if c.core == nil || c.core.State() != "recording" {
		return fault("invalid_state", "Controller", fmt.Errorf("no active drive"))
	}
	return c.advance(milliseconds)
}
func (c *Controller) advance(delta int64) error {
	target := min(c.clock+delta, c.config.StartTimeUnixMS+c.config.DurationMS)
	for {
		if c.pending == nil {
			item, ok := <-c.source
			if !ok {
				return c.fail(fmt.Errorf("mock producer ended without completion"))
			}
			c.pending = &item
		}
		item := c.pending
		if item.done {
			c.pending = nil
			if item.err != nil {
				return c.fail(item.err)
			}
			c.clock = target
			return c.stop()
		}
		if item.record.Timestamp() > target {
			c.clock = target
			return nil
		}
		if err := c.core.Append(item.record); err != nil {
			return c.fail(err)
		}
		c.observe(item.record)
		c.clock = item.record.Timestamp()
		c.pending = nil
	}
}
func (c *Controller) observe(record telemetry.Record) {
	if o := record.Observation; o != nil {
		c.signals[o.SignalID] = *o
		if o.SignalID == "network.tcm_response_latency_ms" && !c.disconnected && c.timeouts["TCM"] == 0 {
			c.modules["TCM"] = "responding"
		}
		return
	}
	e := record.Event
	switch e.EventType {
	case "transport_disconnected":
		c.disconnected = true
		for module := range c.modules {
			c.modules[module] = "transport_unavailable"
		}
		clear(c.timeouts)
	case "transport_connected":
		c.disconnected = false
		for module := range c.modules {
			c.modules[module] = "not_observed"
		}
	case "diagnostic_timeout":
		var attrs struct {
			Expected bool `json:"expected_response"`
		}
		_ = json.Unmarshal(e.Attributes, &attrs)
		if _, ok := c.modules[e.Module]; ok && attrs.Expected && !c.disconnected {
			c.timeouts[e.Module]++
			c.modules[e.Module] = fmt.Sprintf("timeouts: %d", c.timeouts[e.Module])
			if c.timeouts[e.Module] >= c.rules.TimeoutCount {
				c.modules[e.Module] = "unavailable"
			}
		}
	case "response_restored":
		if _, ok := c.modules[e.Module]; ok && !c.disconnected {
			c.timeouts[e.Module] = 0
			c.modules[e.Module] = "responding"
		}
	}
}
func (c *Controller) fail(err error) error {
	wrapFailure("recording", &err)
	c.failed = true
	c.lastError = err.Error()
	c.stopProducer()
	if c.core != nil {
		err = errors.Join(err, c.core.Close())
	}
	return err
}
func (c *Controller) stopProducer() {
	if c.cancel != nil {
		c.cancel()
		<-c.producerDone
		c.cancel = nil
	}
}
func (c *Controller) stop() error {
	c.stopProducer()
	if err := c.core.Stop(c.clock); err != nil {
		return c.fail(err)
	}
	return nil
}
func (c *Controller) Stop() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("Stop", &err)
	if err := c.checkOpen(); err != nil {
		return err
	}
	if c.core == nil || c.core.State() != "recording" {
		return fault("invalid_state", "Controller", fmt.Errorf("no active drive"))
	}
	return c.stop()
}

// Close preserves unfinished metadata if termination prevents a deliberate stop.
func (c *Controller) Close() (err error) {
	defer wrapFailure("Close", &err)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return c.closeErr
	}
	c.closed = true
	c.stopProducer()
	if c.core != nil {
		c.closeErr = c.core.Close()
	}
	return c.closeErr
}
func (c *Controller) Mark(label string) (result string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("Mark", &err)
	if err := c.checkOpen(); err != nil {
		return "", err
	}
	if strings.TrimSpace(label) == "" || len(label) > 256 {
		return "", fault("invalid_input", "Mark", fmt.Errorf("label must contain 1–256 bytes"))
	}
	if c.core == nil || c.core.State() != "recording" {
		return "", fault("invalid_state", "Mark", fmt.Errorf("start a drive before marking an incident"))
	}
	if c.markCount >= maxManualMarks {
		return "", fault("resource_limit", "Mark", fmt.Errorf("manual mark limit reached for this drive; existing evidence preserved"))
	}
	id, err := c.core.MarkIncident(label, c.clock)
	if err == nil {
		c.markCount++
	}
	if err != nil {
		if c.core.State() == "failed" {
			return "", c.fail(err)
		}
		c.lastError = err.Error()
	}
	return id, err
}

func encode(value any) (string, error) {
	data, err := json.Marshal(value)
	if len(data) > maxResponseBytes {
		return "", fault("resource_limit", "encode", fmt.Errorf("bridge response exceeds %d bytes", maxResponseBytes))
	}
	return string(data), err
}
func (c *Controller) Snapshot() (result string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("Snapshot", &err)
	if err := c.checkOpen(); err != nil {
		return "", err
	}
	signals := []telemetry.Observation{}
	for _, original := range c.signals {
		o := original
		if (c.disconnected || c.clock-o.TimestampUnixMS > c.rules.MaxSilenceMS) && o.Quality == "valid" {
			o.Quality = "stale"
		}
		signals = append(signals, o)
	}
	sort.Slice(signals, func(i, j int) bool { return signals[i].SignalID < signals[j].SignalID })
	state := "idle"
	var session *telemetry.Session
	items := []incidents.Incident{}
	if c.core != nil {
		state = c.core.State()
		if c.failed {
			state = "failed"
		}
		value := c.core.Session()
		session = &value
		items = c.core.Incidents()
	}
	elapsed := int64(0)
	if session != nil {
		elapsed = c.clock - session.StartTimeUnixMS
	}
	return encode(map[string]any{"schema_version": 1, "bridge_version": bridgeVersion, "core_version": Version(), "pack_version": c.profile.Manifest.PackVersion, "state": state, "session": session, "signals": signals, "modules": c.modules, "incidents": items, "elapsed_ms": elapsed, "duration_ms": 720000, "logical_time_unix_ms": c.clock, "mock_transport_state": map[bool]string{true: "disconnected", false: "available"}[c.disconnected], "physical_adapter_state": "not_connected", "error": c.lastError})
}

func (c *Controller) sessionPath(id string) (string, error) {
	if strings.ContainsRune(id, 0) || len(id) > 256 || id == "" || id == "." || id == ".." || filepath.Base(id) != id || strings.ContainsAny(id, "/\\") {
		return "", fault("invalid_input", "sessionPath", fmt.Errorf("invalid session ID"))
	}
	path := filepath.Join(c.root, id)
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fault("invalid_input", "sessionPath", fmt.Errorf("session must be a local directory"))
	}
	return path, nil
}
func readSession(path string) (telemetry.Session, error) {
	var session telemetry.Session
	data, err := readDocument(path, "session.json")
	if err != nil {
		return session, err
	}
	if err := decodeDocument(data, &session); err != nil {
		return session, err
	}
	if session.SessionID != filepath.Base(path) {
		return session, fault("corrupt_data", "readSession", fmt.Errorf("directory/session identity mismatch"))
	}
	if err := session.Validate(); err != nil {
		return session, fault("corrupt_data", "readSession", err)
	}
	return session, nil
}
func (c *Controller) ExportPath(id string) (result string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("ExportPath", &err)
	if err := c.checkOpen(); err != nil {
		return "", err
	}
	path, err := c.sessionPath(id)
	if err != nil {
		return "", err
	}
	if c.core != nil && c.core.Session().SessionID == id && c.core.State() != "complete" {
		return "", fault("invalid_state", "ExportPath", fmt.Errorf("stop and finalize the drive before export"))
	}
	session, err := readSession(path)
	if err != nil {
		return "", err
	}
	if session.State != "complete" {
		return "", fault("invalid_state", "ExportPath", fmt.Errorf("incomplete sessions cannot be exported through the app"))
	}
	for _, name := range []string{"session.json", "observations.jsonl", "events.jsonl", "incidents.jsonl", "raw/transport.jsonl"} {
		f, err := openEvidence(path, name, maxLogBytes)
		if err != nil {
			return "", err
		}
		f.Close()
	}
	return path, nil
}

// AdvanceSession rejects a queued refresh from a prior recording generation.
func (c *Controller) AdvanceSession(sessionID string, milliseconds int64) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer wrapFailure("AdvanceSession", &err)
	if err := c.checkOpen(); err != nil {
		return err
	}
	if c.core == nil || c.core.State() != "recording" || c.core.Session().SessionID != sessionID {
		return fault("cancelled", "AdvanceSession", fmt.Errorf("stale or inactive recording generation"))
	}
	if milliseconds < 100 || milliseconds > 10000 || milliseconds%100 != 0 {
		return fault("invalid_input", "AdvanceSession", fmt.Errorf("invalid logical step"))
	}
	return c.advance(milliseconds)
}
