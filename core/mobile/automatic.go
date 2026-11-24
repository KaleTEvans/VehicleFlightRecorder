package mobile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"vehicleflightrecorder.local/core/acquisition"
	"vehicleflightrecorder.local/core/activity"
)

type automaticDisk struct {
	Quiet               bool     `json:"quiet_after_stop,omitempty"`
	ResumeSession       string   `json:"resume_session_id,omitempty"`
	ResumeReason        string   `json:"resume_reason,omitempty"`
	Completion          *endFact `json:"pending_completion,omitempty"`
	LastFailure         string   `json:"last_failure,omitempty"`
	ProtocolProfile     string   `json:"protocol_profile,omitempty"`
	Version             string   `json:"version"`
	Enabled             bool     `json:"enabled"`
	Suppressed          bool     `json:"manual_stop_override"`
	ActiveDrive         string   `json:"active_drive,omitempty"`
	ConfirmationClaimed bool     `json:"confirmation_claimed"`
}
type endFact struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}
type startFact struct {
	SessionID   string `json:"session_id"`
	StartOrigin string `json:"start_origin"`
	TimeMS      int64  `json:"time_unix_ms"`
	VehicleName string `json:"vehicle_name"`
	Coverage    string `json:"coverage"`
}

// AutomaticRecorder owns a single live engine on the transport executor. Probes
// are retained outside Sessions; scanner availability never creates a drive.
// The filesystem lock prevents a second owner. Relaunch preserves interrupted
// originals and always requires new activity evidence for a distinct drive.
type AutomaticRecorder struct {
	ownerID                          string
	interruptedID                    string
	mu                               sync.Mutex
	root, directory, origin, profile string
	lock                             *os.File
	disk                             automaticDisk
	policy                           *activity.Policy
	live                             *LiveRecorder
	state, failure, previous, phase  string
	fact                             *startFact
	lastNS, lastResources            int64
}

func NewAutomaticRecorder(root, origin, profile string) (*AutomaticRecorder, error) {
	if !filepath.IsAbs(root) || (origin != "physical" && origin != "scripted-fixture" && origin != "native-fixture") || (profile != "can11-single-frame-headers-pci" && profile != "can29-normal-fixed-single-frame-headers-pci") {
		return nil, fmt.Errorf("invalid automatic recorder configuration")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	root = resolved
	directory := filepath.Join(root, "Automation")
	if err = os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(directory, "owner.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, fmt.Errorf("automatic owner already exists: %w", err)
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		lock.Close()
		return nil, err
	}
	a := &AutomaticRecorder{ownerID: "owner-" + hex.EncodeToString(nonce[:]), root: root, directory: directory, origin: origin, profile: profile, lock: lock, state: "disabled", phase: "unknown", disk: automaticDisk{Version: activity.Version}}
	data, err := os.ReadFile(filepath.Join(directory, "state.json"))
	if err == nil {
		err = decodeDocument(data, &a.disk)
		if err == nil && a.disk.Version != activity.Version {
			err = fmt.Errorf("unsupported persisted activity policy")
		}
	} else if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		a.Close()
		return nil, err
	}
	if a.disk.ProtocolProfile != "" {
		if a.disk.ProtocolProfile != "can11-single-frame-headers-pci" && a.disk.ProtocolProfile != "can29-normal-fixed-single-frame-headers-pci" {
			a.Close()
			return nil, fmt.Errorf("invalid persisted protocol profile")
		}
		a.profile = a.disk.ProtocolProfile
	}
	if (a.disk.ActiveDrive != "" && !ValidAutomaticSessionID(a.disk.ActiveDrive)) ||
		(a.disk.ResumeSession != "" && !ValidAutomaticSessionID(a.disk.ResumeSession)) ||
		(a.disk.Completion != nil && !ValidAutomaticSessionID(a.disk.Completion.SessionID)) {
		a.Close()
		return nil, fmt.Errorf("invalid persisted drive identity")
	}
	a.disk.ProtocolProfile = a.profile
	a.policy = activity.New(a.disk.Suppressed)
	if a.disk.Enabled && a.disk.Quiet {
		a.state = "awaiting_resume"
		if a.disk.Suppressed {
			a.state = "manual_stop_override"
		}
	}
	if a.disk.LastFailure != "" {
		a.failure = a.disk.LastFailure
		a.state = "failed"
		a.disk.Enabled = false
	}
	// No repair, append, adoption or finalization of the previous evidence files.
	if err = a.journal("owner_opened", map[string]any{"origin": a.origin, "clock_scope": "new_process", "relaunch": "fresh_evidence_new_drive"}); err != nil {
		a.Close()
		return nil, err
	}
	if a.disk.ActiveDrive != "" {
		a.previous = a.disk.ActiveDrive
		a.interruptedID = a.disk.ActiveDrive
		a.disk.ActiveDrive = ""
		a.disk.ConfirmationClaimed = true
		if err = a.journal("interrupted_session_preserved", map[string]any{"session_id": a.previous, "coverage": "unknown_after_last_checkpoint", "relaunch_policy": "fresh_evidence_new_drive"}); err != nil {
			a.Close()
			return nil, err
		}
	}
	if err = a.persist(); err != nil {
		a.Close()
		return nil, err
	}
	return a, nil
}
func (a *AutomaticRecorder) persist() error {
	a.disk.Suppressed = a.policy.Suppressed
	data, err := json.Marshal(a.disk)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(a.directory, ".state-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		var n int
		n, err = f.Write(append(data, '\n'))
		if err == nil && n != len(data)+1 {
			err = io.ErrShortWrite
		}
	}
	if err == nil {
		err = f.Sync()
	}
	c := f.Close()
	if err == nil {
		err = c
	}
	if err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(a.directory, "state.json"))
}
func (a *AutomaticRecorder) journal(kind string, details any) error {
	f, err := os.OpenFile(filepath.Join(a.directory, "journal.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() > 16<<20 {
		return fmt.Errorf("automation journal storage limit; originals preserved")
	}
	b, err := json.Marshal(map[string]any{"policy_version": activity.Version, "owner_id": a.ownerID, "origin": a.origin, "observed_wall_unix_ms": time.Now().UnixMilli(), "event": kind, "native_ns": a.lastNS, "phase": a.phase, "details": details})
	if err != nil {
		return err
	}
	if len(b) > 8192 {
		return fmt.Errorf("automation event budget")
	}
	n, err := f.Write(append(b, '\n'))
	if err == nil && n != len(b)+1 {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = f.Sync()
	}
	return err
}
func (a *AutomaticRecorder) fail(err error) error {
	a.failure = err.Error()
	if len(a.failure) > 512 {
		a.failure = a.failure[:512]
	}
	a.disk.LastFailure = a.failure
	a.state = "failed"
	a.fact = nil
	a.disk.Enabled = false
	if a.live != nil {
		a.live.mu.Lock()
		a.live.fail(err)
		a.live.mu.Unlock()
	}
	_ = a.journal("monitoring_failed", map[string]any{"failure": a.failure, "evidence": "preserved_incomplete", "recovery": "explicit_rearm_after_adapter_ready"})
	_ = a.persist()
	return err
}
func (a *AutomaticRecorder) create(purpose string, wall, now int64) error {
	return a.createSegment(purpose, wall, now, "")
}

func (a *AutomaticRecorder) createSegment(purpose string, wall, now int64, parent string) error {
	directory := filepath.Join(a.root, "Sessions")
	workload := "combined"
	if purpose == "activity_probe" {
		directory = filepath.Join(a.directory, "Probes")
		workload = "rpm_only"
	}
	options, _ := json.Marshal(liveOptions{Origin: a.origin, OperatingState: "unknown", Workload: workload, ProtocolProfile: a.profile, StartOrigin: purpose, PolicyVersion: activity.Version, ResumedFrom: parent})
	l, err := NewLiveRecorder(directory, string(options), wall, now)
	if err != nil {
		return err
	}
	a.live = l
	a.fact = nil
	if purpose == "automatic" {
		a.disk.ActiveDrive = filepath.Base(l.path)
		a.disk.ConfirmationClaimed = false
		if err = a.persist(); err != nil {
			return err
		}
		return a.live.runtimeEvent(now, "automatic_session_initializing", map[string]any{"policy": a.policy.Config, "pre_start_context": a.previous, "context_scope": "separate_retained_probe_not_drive_telemetry", "initial_coverage": "unknown_until_fresh_reads"})
	}
	return nil
}
func (a *AutomaticRecorder) SetEnabled(enabled bool, wallMS, nativeNS int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lock == nil {
		return fmt.Errorf("automatic recorder closed")
	}
	if nativeNS < a.lastNS {
		return fmt.Errorf("automatic clock regression")
	}
	a.lastNS = nativeNS
	if enabled && a.disk.Enabled && a.failure == "" && (a.live != nil || a.disk.Quiet) {
		return nil
	}
	previousFailure := a.failure
	if a.live != nil {
		a.live.mu.Lock()
		incomplete := a.live.failed || a.live.core.State() == "failed"
		a.live.mu.Unlock()
		if !incomplete {
			if err := a.live.Stop(nativeNS); err != nil {
				previousFailure = err.Error()
				incomplete = true
			}
		}
		if incomplete {
			a.live.mu.Lock()
			_ = a.live.core.Close()
			a.live.mu.Unlock()
		}
		a.live = nil
	}
	// Disarming remains available after acquisition failure. Successful explicit
	// rearm still has to create/checkpoint a new engine and persist its state.
	a.failure = ""
	a.disk.LastFailure = ""
	if previousFailure != "" {
		if err := a.journal("failed_capture_preserved", map[string]any{"failure": previousFailure, "action": "explicit_disarm_or_rearm"}); err != nil {
			return a.fail(err)
		}
	}
	a.disk.Enabled = enabled
	a.disk.ActiveDrive = ""
	a.fact = nil
	a.policy = activity.New(false)
	a.state = "disabled"
	if enabled {
		a.disk.Quiet = false
		a.disk.ResumeSession = ""
		a.disk.ResumeReason = ""
		a.state = "waiting_for_activity"
		if err := a.create("activity_probe", wallMS, nativeNS); err != nil {
			return a.fail(err)
		}
	}
	if err := a.journal("automatic_preference_changed", map[string]any{"enabled": enabled}); err != nil {
		return a.fail(err)
	}
	if err := a.persist(); err != nil {
		return a.fail(err)
	}
	return nil
}
func (a *AutomaticRecorder) Tick(nativeNS, wallMS int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lock == nil {
		return fmt.Errorf("automatic recorder closed")
	}
	if nativeNS < a.lastNS {
		return a.fail(fmt.Errorf("automatic clock regression"))
	}
	a.lastNS = nativeNS
	if !a.disk.Enabled || a.disk.Quiet || a.failure != "" {
		return nil
	}
	if a.live == nil {
		if err := a.create("activity_probe", wallMS, nativeNS); err != nil {
			return a.fail(err)
		}
	}
	l := a.live
	l.mu.Lock()
	complete := l.core.State() == "complete"
	failed := l.failed
	failureReason := l.failure
	connected := l.connected && l.ready
	var evidence *activity.Evidence
	if transaction, clean := l.scheduler.LatestRPMResult(); clean {
		for _, o := range l.cached {
			if o.Signal != "powertrain.engine.rpm" || o.Transaction != transaction {
				continue
			}
			if evidence == nil {
				evidence = &activity.Evidence{Transaction: o.Transaction, Generation: o.Generation, ReceiveNS: o.ReceiveNS + l.baseNS}
			}
			evidence.Responders++
			if o.Value > evidence.MaximumRPM {
				evidence.MaximumRPM = o.Value
			}
		}
	}
	observations := l.observationCursor
	initialized := l.scheduler.Initialized()
	purpose := l.options.StartOrigin
	l.mu.Unlock()
	if failed {
		return a.fail(fmt.Errorf("%s; original evidence preserved", failureReason))
	}
	if complete {
		if purpose == "automatic" {
			return a.finishDrive(l, nativeNS, "controlled_recording_duration_limit")
		}
		a.previous = filepath.Base(l.path)
		a.disk.ActiveDrive = ""
		a.fact = nil
		if err := a.journal("bounded_session_ended", map[string]any{"session_id": a.previous, "purpose": purpose, "maximum_seconds": 300}); err != nil {
			return a.fail(err)
		}
		if err := a.create("activity_probe", wallMS, nativeNS); err != nil {
			return a.fail(err)
		}
		if err := a.persist(); err != nil {
			return a.fail(err)
		}
		return nil
	}
	beforeSuppressed := a.policy.Suppressed
	decision, err := a.policy.Step(nativeNS, evidence, connected)
	if err != nil {
		return a.fail(err)
	}
	l.mu.Lock()
	l.rpmRecovery = purpose == "automatic" && (decision.State == "degraded_gap" || decision.State == "end_grace")
	l.mu.Unlock()
	if decision.State != a.state {
		a.state = decision.State
		if err = a.journal("activity_transition", map[string]any{"state": a.state, "reason": decision.Reason, "evidence": evidence}); err != nil {
			return a.fail(err)
		}
		if purpose == "automatic" {
			if err = l.runtimeEvent(nativeNS, "automatic_activity_transition", map[string]any{"state": a.state, "reason": decision.Reason, "evidence": evidence}); err != nil {
				return a.fail(err)
			}
		}
	}
	if beforeSuppressed != a.policy.Suppressed {
		if err = a.persist(); err != nil {
			return a.fail(err)
		}
	}
	if decision.Action == "start" {
		a.previous = filepath.Base(l.path)
		if err = l.Stop(nativeNS); err != nil {
			return a.fail(err)
		}
		if err = a.create("automatic", wallMS, nativeNS); err != nil {
			return a.fail(err)
		}
		a.policy = activity.New(false)
		a.policy.Recording = true
		a.state = "starting"
	} else if decision.Action == "end" {
		return a.finishDrive(l, nativeNS, decision.Reason)
	} else if purpose == "automatic" && observations > 0 && initialized && a.fact == nil && !a.disk.ConfirmationClaimed {
		coverage := "limited_standard_obd" // no verified TCM / full vehicle coverage claim
		a.fact = &startFact{filepath.Base(l.path), "automatic", l.anchorMS + (nativeNS-l.baseNS)/1_000_000, "Vehicle", coverage}
		if err = l.runtimeEvent(nativeNS, "automatic_recording_started", a.fact); err != nil {
			return a.fail(err)
		}
		if err = l.sync(); err != nil {
			return a.fail(err)
		}
	}
	if a.lastResources == 0 || nativeNS-a.lastResources >= 60*acquisition.Second {
		a.lastResources = nativeNS
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		var total int64
		count := 0
		err = filepath.WalkDir(a.root, func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			count++
			if count > 20000 {
				return fmt.Errorf("evidence directory count limit")
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink in evidence storage")
			}
			if !d.IsDir() {
				info, e := d.Info()
				if e != nil {
					return e
				}
				total += info.Size()
			}
			return nil
		})
		if err != nil {
			return a.fail(err)
		}
		if total > 512<<20 {
			return a.fail(fmt.Errorf("512 MiB evidence limit; export existing files, no evidence deleted"))
		}
		if err = a.journal("resource_sample", map[string]any{"retained_bytes": total, "go_heap_alloc_bytes": memory.HeapAlloc, "go_heap_objects": memory.HeapObjects, "maximum_drive_seconds": 1800}); err != nil {
			return a.fail(err)
		}
	}
	return nil
}

// Freeze transport rearm durably before finalizing the original.
func (a *AutomaticRecorder) finishDrive(l *LiveRecorder, now int64, reason string) error {
	a.disk.Quiet = true
	if err := a.persist(); err != nil {
		return a.fail(err)
	}
	l.mu.Lock()
	recording := l.core.State() == "recording"
	l.mu.Unlock()
	if recording {
		if err := l.runtimeEvent(now, "automatic_session_end", map[string]any{"reason": reason}); err != nil {
			return a.fail(err)
		}
		if err := l.stopAt(now, reason); err != nil {
			return a.fail(err)
		}
	}
	id := filepath.Base(l.path)
	a.live, a.fact = nil, nil
	a.previous = id
	a.state = "awaiting_resume"
	a.disk.ActiveDrive = ""
	a.disk.ResumeSession, a.disk.ResumeReason = id, reason
	a.disk.Completion = &endFact{id, reason}
	if err := a.persist(); err != nil {
		return a.fail(err)
	}
	return nil
}

// Resume only the currently eligible saved drive, in a separate capture.
func (a *AutomaticRecorder) Resume(sessionID string, wallMS, nativeNS int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lock == nil || nativeNS < a.lastNS || a.failure != "" || !a.disk.Enabled ||
		!a.disk.Quiet || !ValidAutomaticSessionID(sessionID) || a.disk.ResumeSession != sessionID {
		return fmt.Errorf("saved drive is no longer available to resume")
	}
	a.lastNS = nativeNS
	a.policy = activity.New(false)
	a.policy.Recording = true
	a.disk.Quiet = false
	a.disk.ResumeSession, a.disk.ResumeReason = "", ""
	a.state = "starting"
	if err := a.createSegment("automatic", wallMS, nativeNS, sessionID); err != nil {
		return a.fail(err)
	}
	return nil
}

// Consume the persisted completion before native feedback can be submitted.
func (a *AutomaticRecorder) ClaimCompletion() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lock == nil || a.disk.Completion == nil {
		return "", nil
	}
	payload, err := json.Marshal(a.disk.Completion)
	if err != nil {
		return "", err
	}
	a.disk.Completion = nil
	if err := a.persist(); err != nil {
		return "", a.fail(err)
	}
	return string(payload), nil
}

func (l *LiveRecorder) runtimeEvent(nativeNS int64, kind string, attributes any) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.failed || l.core.State() != "recording" {
		return fmt.Errorf("runtime event needs active session")
	}
	if _, err := l.clock(nativeNS); err != nil {
		return err
	}
	return l.event(kind, "info", attributes)
}
func (l *LiveRecorder) sync() error              { l.mu.Lock(); defer l.mu.Unlock(); return l.core.Sync() }
func (a *AutomaticRecorder) Live() *LiveRecorder { a.mu.Lock(); defer a.mu.Unlock(); return a.live }
func (a *AutomaticRecorder) Stop(nativeNS, wallMS int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lock == nil {
		return fmt.Errorf("automatic recorder closed")
	}

	if nativeNS < a.lastNS {
		return fmt.Errorf("automatic clock regression")
	}
	a.lastNS = nativeNS
	if a.live == nil {
		return nil
	}
	if err := a.live.Stop(nativeNS); err != nil {
		return a.fail(err)
	}
	a.policy.ManualStop()
	a.disk.ActiveDrive = ""
	a.fact = nil
	a.state = "manual_stop_override"
	if err := a.journal("manual_stop_override", map[string]any{"until": "explicit_rearm"}); err != nil {
		return a.fail(err)
	}
	a.live = nil
	a.disk.Quiet = true
	if err := a.persist(); err != nil {
		return a.fail(err)
	}
	return nil
}

// Claim before any native side effect. A crash after this durable claim may miss
// one confirmation; it can never resend on relaunch or a new transport generation.
func (a *AutomaticRecorder) ClaimConfirmation(nativeNS int64) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lock == nil {
		return "", fmt.Errorf("automatic recorder closed")
	}

	if nativeNS < a.lastNS {
		return "", fmt.Errorf("automatic clock regression")
	}
	a.lastNS = nativeNS
	if a.fact == nil || a.disk.ConfirmationClaimed {
		return "", nil
	}
	a.disk.ConfirmationClaimed = true
	if err := a.persist(); err != nil {
		return "", a.fail(err)
	}
	if err := a.journal("confirmation_claimed", a.fact); err != nil {
		return "", a.fail(err)
	}
	b, err := json.Marshal(a.fact)
	return string(b), err
}
func (a *AutomaticRecorder) NativeEvent(kind, detail string, nativeNS int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lock == nil {
		return fmt.Errorf("automatic recorder closed")
	}

	if len(detail) > 2048 {
		return fmt.Errorf("native automation detail budget")
	}
	if nativeNS < a.lastNS {
		return fmt.Errorf("automatic clock regression")
	}
	a.lastNS = nativeNS
	switch kind {
	case "foreground", "background", "inactive":
		a.phase = kind
	case "battery_sample":
		if !a.disk.Enabled {
			return nil
		}
	case "confirmation_result", "notification_tap", "accessory_directory", "export_requested", "export_result":
	default:
		return fmt.Errorf("unknown automatic native event")
	}
	if err := a.journal(kind, map[string]any{"detail": detail}); err != nil {
		return a.fail(err)
	}
	if a.live != nil && a.disk.ActiveDrive != "" {
		a.live.mu.Lock()
		active := !a.live.failed && a.live.core.State() == "recording"
		a.live.mu.Unlock()
		if !active {
			return nil
		}
		return a.live.runtimeEvent(nativeNS, "automatic_"+kind, map[string]any{"detail": detail})
	}
	return nil
}
func (a *AutomaticRecorder) Snapshot() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	snapshot := map[string]any{"policy": a.policy.Config, "enabled": a.disk.Enabled, "state": a.state, "error": a.failure, "session_id": a.disk.ActiveDrive, "accepting_telemetry": a.fact != nil && a.disk.ActiveDrive != "", "manual_stop_override": a.policy.Suppressed, "previous_interrupted_session": a.interruptedID, "previous_context_capture": a.previous, "phase": a.phase, "maximum_drive_seconds": 1800, "quiet": a.disk.Quiet}
	if a.disk.ResumeSession != "" {
		snapshot["resume_session_id"] = a.disk.ResumeSession
		snapshot["resume_reason"] = a.disk.ResumeReason
	}
	b, _ := json.Marshal(snapshot)
	return string(b)
}
func (a *AutomaticRecorder) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lock == nil {
		return nil
	}
	if a.live != nil {
		a.live.mu.Lock()
		_ = a.live.core.Close()
		a.live.mu.Unlock()
	}
	err := syscall.Flock(int(a.lock.Fd()), syscall.LOCK_UN)
	c := a.lock.Close()
	a.lock = nil
	if err != nil {
		return err
	}
	return c
}

// Avoid accidental native calls to path-driven commands: only existing session IDs
// are used for navigation. No new drive can be created by a notification tap.
func ValidAutomaticSessionID(id string) bool {
	if !strings.HasPrefix(id, "live-") || len(id) != 37 || id != strings.ToLower(id) {
		return false
	}
	_, err := hex.DecodeString(id[5:])
	return err == nil
}

// Profile selection changes are explicit and allowed only while disarmed.
func (a *AutomaticRecorder) SetProfile(profile string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.disk.Enabled || a.lock == nil {
		return fmt.Errorf("disable monitoring before selecting its fixed profile")
	}
	if profile != "can11-single-frame-headers-pci" && profile != "can29-normal-fixed-single-frame-headers-pci" {
		return fmt.Errorf("unknown automatic profile")
	}
	a.profile = profile
	a.disk.ProtocolProfile = profile
	return a.persist()
}
