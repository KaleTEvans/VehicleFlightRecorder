package acquisition

import (
	"bytes"
	"fmt"
	"strings"
)

// StartupSyncVersion bounds old-stream settling without interpreting residual
// bytes as an acknowledgement or permitting vehicle reads before full setup.
const StartupSyncVersion = "quiet-start-1"

func (s *Scheduler) BeginStartup(now int64) error {
	if err := s.clock(now); err != nil {
		return err
	}
	if s.report.Configuration == nil || s.report.Configuration.StartupSyncVersion == "" {
		return nil
	}
	s.settling = true
	s.quietUntil = now + Second
	s.settleLimit = now + 3*Second
	s.stoppedRetries = 0
	return nil
}
func (s *Scheduler) captureSettling(now int64, data []byte) error {
	if err := s.clock(now); err != nil {
		return err
	}
	if err := s.raw("unsolicited", "", now, data); err != nil {
		return err
	}
	// Keep the original bytes; they are not attributed to any new request.
	if now >= s.settleLimit {
		s.report.State = "blocked"
		s.issue("startup_quiet_deadline")
		return fmt.Errorf("startup stream did not become quiet")
	}
	if quiet := now + Second/2; quiet > s.quietUntil {
		s.quietUntil = quiet
	}
	return nil
}
func stoppedClose(frame []byte) bool {
	if len(frame) > 256 || len(frame) < 2 || frame[len(frame)-1] != '>' || bytes.Count(frame, []byte{'>'}) != 1 {
		return false
	}
	lines := strings.FieldsFunc(string(frame[:len(frame)-1]), func(r rune) bool { return r == '\r' || r == '\n' })
	clean := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "STPC" {
			continue
		}
		clean = append(clean, line)
	}
	return len(clean) == 1 && clean[0] == "STOPPED"
}
