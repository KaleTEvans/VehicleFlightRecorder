package recorder

import (
	"fmt"
	"time"

	"vehicleflightrecorder.local/core/telemetry"
)

// Buffer uses an inclusive cutoff [latest-window, latest]. It is single-owner,
// rejects backwards time, and owns copies of all records and snapshots.
type Buffer struct {
	windowMS int64
	latest   int64
	records  []telemetry.Record
}

func NewBuffer(window time.Duration) (*Buffer, error) {
	if window < time.Millisecond || window%time.Millisecond != 0 {
		return nil, fmt.Errorf("buffer window must be positive whole milliseconds")
	}
	return &Buffer{windowMS: window.Milliseconds()}, nil
}

func (b *Buffer) check(r telemetry.Record) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Timestamp() < b.latest {
		return fmt.Errorf("out-of-order timestamp %d before %d", r.Timestamp(), b.latest)
	}
	return nil
}

func (b *Buffer) Add(r telemetry.Record) error {
	if err := b.check(r); err != nil {
		return err
	}
	b.records = append(b.records, r.Clone())
	return b.Advance(r.Timestamp())
}

func (b *Buffer) Advance(timestamp int64) error {
	if timestamp <= 0 || timestamp < b.latest {
		return fmt.Errorf("buffer time must be positive and nondecreasing")
	}
	b.latest = timestamp
	cutoff := timestamp - b.windowMS
	index := 0
	for index < len(b.records) && b.records[index].Timestamp() < cutoff {
		index++
	}
	if index == 0 {
		return nil
	}
	// Clear references and compact so expired samples do not retain memory.
	copy(b.records, b.records[index:])
	clear(b.records[len(b.records)-index:])
	b.records = b.records[:len(b.records)-index]
	return nil
}

func (b *Buffer) Snapshot() []telemetry.Record {
	result := make([]telemetry.Record, len(b.records))
	for i, record := range b.records {
		result[i] = record.Clone()
	}
	return result
}
