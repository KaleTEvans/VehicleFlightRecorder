package incidents

import (
	"fmt"
	"math"

	"vehicleflightrecorder.local/core/telemetry"
)

type Notice struct {
	IncidentID string
	Trigger    Trigger
}
type Result struct {
	Updates   []Incident
	Triggers  []Notice
	Finalized []string
}

type Engine struct {
	config         Config
	sessionID      string
	start          int64
	latest         int64
	detector       Detector
	items          []Incident
	gaps           []Gap
	disconnectedAt *int64
	nextTrigger    int
	closed         bool
}

func New(config Config, sessionID string, start int64) (*Engine, error) {
	if sessionID == "" || start < 1 {
		return nil, fmt.Errorf("incident engine requires session identity/start")
	}
	detector, err := NewDetector(config)
	if err != nil {
		return nil, err
	}
	return &Engine{config: config, sessionID: sessionID, start: start, detector: detector}, nil
}

// Process borrows buffer for this call only; it never mutates or retains records.
func (e *Engine) Process(record telemetry.Record, reference Reference, buffer []telemetry.Record) (Result, error) {
	if e.closed {
		return Result{}, fmt.Errorf("incident engine is closed")
	}
	if err := record.Validate(); err != nil {
		return Result{}, err
	}
	if err := reference.Validate(); err != nil {
		return Result{}, err
	}
	timestamp := record.Timestamp()
	if record.SessionID() != e.sessionID || timestamp < e.start || timestamp < e.latest || reference.TimestampUnixMS != timestamp {
		return Result{}, fmt.Errorf("incident record identity/time mismatch")
	}
	if reference.RecordSequence != record.Sequence() || (record.Observation != nil && (reference.File != "observations.jsonl" || reference.SignalID != record.Observation.SignalID)) || (record.Event != nil && (reference.File != "events.jsonl" || reference.EventType != record.Event.EventType)) {
		return Result{}, fmt.Errorf("incident evidence reference does not match record")
	}
	wasDisconnected := e.disconnectedAt != nil
	gapChanged := false
	if e.latest > 0 && timestamp-e.latest > e.config.MaxSilenceMS && !wasDisconnected {
		e.gaps = append(e.gaps, Gap{StartTimeUnixMS: e.latest + e.config.MaxSilenceMS, EndTimeUnixMS: timestamp, Reason: "record_silence"})
		gapChanged = true
	}
	if event := record.Event; event != nil {
		if event.EventType == "transport_disconnected" && e.disconnectedAt == nil {
			begin := timestamp
			e.disconnectedAt = &begin
			gapChanged = true
		}
		if event.EventType == "transport_connected" && e.disconnectedAt != nil {
			if timestamp > *e.disconnectedAt {
				e.gaps = append(e.gaps, Gap{StartTimeUnixMS: *e.disconnectedAt, EndTimeUnixMS: timestamp, Reason: "transport_disconnected"})
			}
			e.disconnectedAt = nil
			gapChanged = true
		}
	}
	e.latest = timestamp
	candidates, err := e.detector.Evaluate(record)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	dirty := map[int]bool{}
	for index := range e.items {
		if e.items[index].Status == "capturing" && e.items[index].WindowEndUnixMS < timestamp {
			e.refresh(index, timestamp, true)
			dirty[index] = true
			result.Finalized = append(result.Finalized, e.items[index].IncidentID)
		}
	}
	for _, candidate := range candidates {
		if timestamp > math.MaxInt64-e.config.PostWindowMS {
			return Result{}, fmt.Errorf("incident post-window overflows timestamp")
		}
		requestedStart, requestedEnd := timestamp-e.config.PreWindowMS, timestamp+e.config.PostWindowMS
		index := -1
		for n := range e.items {
			if e.items[n].Status == "capturing" && requestedStart <= e.items[n].WindowEndUnixMS && requestedEnd >= e.items[n].WindowStartUnixMS {
				index = n
				break
			}
		}
		if index < 0 {
			availableStart := max(requestedStart, e.start)
			for _, sample := range buffer {
				if sample.Timestamp() >= requestedStart {
					availableStart = max(availableStart, sample.Timestamp())
					break
				}
			}
			item := Incident{SchemaVersion: 1, IncidentID: fmt.Sprintf("%s.incident-%06d", e.sessionID, len(e.items)+1), SessionID: e.sessionID, TriggerTimeUnixMS: timestamp, TriggerType: candidate.Type, PreWindowSeconds: float64(e.config.PreWindowMS) / 1000, PostWindowSeconds: float64(e.config.PostWindowMS) / 1000, WindowStartUnixMS: requestedStart, WindowEndUnixMS: requestedEnd, AvailableStartUnixMS: availableStart, AvailableEndUnixMS: timestamp, Status: "capturing", Severity: "info", Triggers: []Trigger{}, RelatedIncidentIDs: []string{}, CoverageGaps: []Gap{}, IncompleteReasons: []string{}}
			index = len(e.items)
			for n := range e.items {
				old := &e.items[n]
				if requestedStart <= old.WindowEndUnixMS && requestedEnd >= old.WindowStartUnixMS {
					item.RelatedIncidentIDs = append(item.RelatedIncidentIDs, old.IncidentID)
					old.RelatedIncidentIDs = append(old.RelatedIncidentIDs, item.IncidentID)
					dirty[n] = true
				}
			}
			e.items = append(e.items, item)
		}
		e.nextTrigger++
		trigger := Trigger{TriggerID: fmt.Sprintf("trigger-%06d", e.nextTrigger), Type: candidate.Type, TimestampUnixMS: timestamp, Severity: candidate.Severity, Category: candidate.Category, Module: candidate.Module, Manual: candidate.Manual, Label: candidate.Label, Evidence: reference, Details: append([]byte(nil), candidate.Details...)}
		item := &e.items[index]
		item.Triggers = append(item.Triggers, trigger)
		item.WindowEndUnixMS = max(item.WindowEndUnixMS, requestedEnd)
		item.Manual = item.Manual || trigger.Manual
		if severityRank(trigger.Severity) > severityRank(item.Severity) {
			item.Severity = trigger.Severity
		}
		dirty[index] = true
		result.Triggers = append(result.Triggers, Notice{IncidentID: item.IncidentID, Trigger: trigger})
	}
	for index := range e.items {
		item := &e.items[index]
		if item.Status == "capturing" {
			finished := timestamp >= item.WindowEndUnixMS
			e.refresh(index, timestamp, finished)
			if finished {
				dirty[index] = true
				result.Finalized = append(result.Finalized, item.IncidentID)
			} else if gapChanged {
				dirty[index] = true
			}
		}
		if dirty[index] {
			item.Revision++
			result.Updates = append(result.Updates, item.Clone())
		}
	}
	return result, nil
}

func (e *Engine) refresh(index int, timestamp int64, finished bool) {
	item := &e.items[index]
	item.AvailableEndUnixMS = min(timestamp, item.WindowEndUnixMS)
	item.CoverageGaps = []Gap{}
	appendGap := func(gap Gap) {
		gap.StartTimeUnixMS = max(gap.StartTimeUnixMS, item.AvailableStartUnixMS)
		gap.EndTimeUnixMS = min(gap.EndTimeUnixMS, item.AvailableEndUnixMS)
		if gap.EndTimeUnixMS > gap.StartTimeUnixMS {
			item.CoverageGaps = append(item.CoverageGaps, gap)
		}
	}
	for _, gap := range e.gaps {
		appendGap(gap)
	}
	if e.disconnectedAt != nil {
		appendGap(Gap{StartTimeUnixMS: *e.disconnectedAt, EndTimeUnixMS: timestamp, Reason: "transport_disconnected"})
	}
	item.IncompleteReasons = []string{}
	if item.AvailableStartUnixMS > item.WindowStartUnixMS {
		item.IncompleteReasons = append(item.IncompleteReasons, "pre_window_unavailable")
	}
	if item.AvailableEndUnixMS < item.WindowEndUnixMS {
		item.IncompleteReasons = append(item.IncompleteReasons, "post_window_unavailable")
	}
	if len(item.CoverageGaps) > 0 {
		item.IncompleteReasons = append(item.IncompleteReasons, "coverage_gap")
	}
	if finished {
		item.Status = "captured"
		if len(item.IncompleteReasons) > 0 {
			item.Status = "partial"
		}
	}
}

func (e *Engine) Finish(timestamp int64) (Result, error) {
	if e.closed || timestamp < e.latest {
		return Result{}, fmt.Errorf("invalid incident finalization")
	}
	result := Result{}
	for index := range e.items {
		if e.items[index].Status == "capturing" {
			e.refresh(index, timestamp, true)
			e.items[index].Revision++
			result.Updates = append(result.Updates, e.items[index].Clone())
			result.Finalized = append(result.Finalized, e.items[index].IncidentID)
		}
	}
	e.closed = true
	return result, nil
}

func (e *Engine) Count() int { return len(e.items) }
func (e *Engine) Snapshot() []Incident {
	items := make([]Incident, len(e.items))
	for n := range e.items {
		items[n] = e.items[n].Clone()
	}
	return items
}
