package acquisition

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func clone(r Report) Report {
	b, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	var out Report
	if err = json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}

// WriteReport never overwrites a prior report. This version is a synthetic test
// artifact, not the canonical live-drive schema or proof of physical capability.
func (s *Scheduler) WriteReport(path string) error {
	r := s.Report()
	if r.Version != 1 || r.Kind != "offline-acquisition" || r.Source != "synthetic" {
		return fmt.Errorf("unsupported report contract")
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if len(b) > 8<<20 {
		return fmt.Errorf("report exceeds 8 MiB")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	n, err := f.Write(b)
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = f.Sync()
	}
	e := f.Close()
	if err == nil {
		err = e
	}
	return err
}
