package mobile

import (
	"context"
	"encoding/json"
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
}

func NewController(packDirectory, sessionsDirectory string) (*Controller, error) {
	if !filepath.IsAbs(packDirectory) || !filepath.IsAbs(sessionsDirectory) {
		return nil, fmt.Errorf("pack and sessions directories must be absolute")
	}
	p, err := vehicle.Load(packDirectory)
	if err != nil {
		return nil, err
	}
	rules, err := vehicle.LoadIncidentConfig(packDirectory, p)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(sessionsDirectory, 0700); err != nil {
		return nil, err
	}
	return &Controller{pack: packDirectory, root: sessionsDirectory, profile: p, rules: rules, signals: map[string]telemetry.Observation{}, modules: map[string]string{}}, nil
}

func (c *Controller) Start(scenarioName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.core != nil && c.core.State() == "recording" {
		return fmt.Errorf("a drive is already recording")
	}
	if c.core != nil && c.core.State() == "failed" {
		return fmt.Errorf("recorder failed; restart the controller before starting another drive")
	}
	var scenario *vehicle.Scenario
	if scenarioName != "healthy_drive" {
		value, err := vehicle.LoadScenario(c.pack, scenarioName, c.profile)
		if err != nil {
			return err
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
func (c *Controller) Advance(milliseconds int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if milliseconds < 100 || milliseconds > 10000 || milliseconds%100 != 0 {
		return fmt.Errorf("advance requires 100–10000 ms in 100 ms steps")
	}
	if c.core == nil || c.core.State() != "recording" {
		return fmt.Errorf("no active drive")
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
	c.lastError = err.Error()
	c.stopProducer()
	if c.core != nil {
		_ = c.core.Close()
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
func (c *Controller) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.core == nil || c.core.State() != "recording" {
		return fmt.Errorf("no active drive")
	}
	return c.stop()
}

// Close preserves unfinished metadata if termination prevents a deliberate stop.
func (c *Controller) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopProducer()
	if c.core != nil {
		return c.core.Close()
	}
	return nil
}
func (c *Controller) Mark(label string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.core == nil || c.core.State() != "recording" {
		return "", fmt.Errorf("start a drive before marking an incident")
	}
	id, err := c.core.MarkIncident(label, c.clock)
	if err != nil {
		c.lastError = err.Error()
	}
	return id, err
}

func encode(value any) (string, error) { data, err := json.Marshal(value); return string(data), err }
func (c *Controller) Snapshot() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
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
		value := c.core.Session()
		session = &value
		items = c.core.Incidents()
	}
	elapsed := int64(0)
	if session != nil {
		elapsed = c.clock - session.StartTimeUnixMS
	}
	return encode(map[string]any{"schema_version": 1, "core_version": Version(), "pack_version": c.profile.Manifest.PackVersion, "state": state, "session": session, "signals": signals, "modules": c.modules, "incidents": items, "elapsed_ms": elapsed, "duration_ms": 720000, "logical_time_unix_ms": c.clock, "mock_transport_state": map[bool]string{true: "disconnected", false: "available"}[c.disconnected], "physical_adapter_state": "not_connected", "error": c.lastError})
}

func (c *Controller) sessionPath(id string) (string, error) {
	if id == "" || id == "." || id == ".." || filepath.Base(id) != id || strings.ContainsAny(id, "/\\") {
		return "", fmt.Errorf("invalid session ID")
	}
	path := filepath.Join(c.root, id)
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("session must be a local directory")
	}
	return path, nil
}
func readSession(path string) (telemetry.Session, error) {
	var session telemetry.Session
	data, err := os.ReadFile(filepath.Join(path, "session.json"))
	if err != nil {
		return session, err
	}
	if err := json.Unmarshal(data, &session); err != nil {
		return session, err
	}
	return session, session.Validate()
}
func (c *Controller) SessionsJSON() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries, err := os.ReadDir(c.root)
	if err != nil {
		return "", err
	}
	rows := []map[string]any{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path, err := c.sessionPath(entry.Name())
		if err != nil {
			return "", err
		}
		session, err := readSession(path)
		if err != nil {
			return "", fmt.Errorf("session %s: %w", entry.Name(), err)
		}
		items, err := readIncidents(path)
		if err != nil {
			return "", err
		}
		if c.core != nil && session.SessionID == c.core.Session().SessionID {
			session = c.core.Session()
			items = c.core.Incidents()
		}
		if session.State == "recording" && (c.core == nil || session.SessionID != c.core.Session().SessionID) {
			counts := map[string]int64{}
			for _, name := range []string{"observations.jsonl", "events.jsonl"} {
				if err := scanJSON(path, name, func(_ int64, _ json.RawMessage) error { counts[name]++; return nil }); err != nil {
					return "", err
				}
			}
			session.ObservationCount = counts["observations.jsonl"]
			session.EventCount = counts["events.jsonl"]
			session.IncidentCount = int64(len(items))
		}
		var size int64
		err = filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				info, err := d.Info()
				if err != nil {
					return err
				}
				size += info.Size()
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		rows = append(rows, map[string]any{"session": session, "incidents": items, "size_bytes": size})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i]["session"].(telemetry.Session).StartTimeUnixMS > rows[j]["session"].(telemetry.Session).StartTimeUnixMS
	})
	return encode(rows)
}
func (c *Controller) ExportPath(id string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	path, err := c.sessionPath(id)
	if err != nil {
		return "", err
	}
	if c.core != nil && c.core.Session().SessionID == id && c.core.State() != "complete" {
		return "", fmt.Errorf("stop and finalize the drive before export")
	}
	session, err := readSession(path)
	if err != nil {
		return "", err
	}
	if session.State != "complete" {
		return "", fmt.Errorf("incomplete sessions cannot be exported through the app")
	}
	return path, nil
}
