package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCommandReproducibleFilesAndNoOverwrite(t *testing.T) {
	root := t.TempDir()
	pack := "../../../vehicle-packs/alfa-giulia-qv-2017"
	first, second := filepath.Join(root, "a"), filepath.Join(root, "b")
	var stdout, stderr bytes.Buffer
	for _, out := range []string{first, second} {
		if err := run([]string{"--pack", pack, "--output", out, "--duration", "2m"}, &stdout, &stderr); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"session.json", "observations.jsonl", "events.jsonl", "incidents.jsonl", "raw/transport.jsonl"} {
		a, err := os.ReadFile(filepath.Join(first, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(second, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("nondeterministic output %s", name)
		}
	}
	before, err := os.ReadFile(filepath.Join(first, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if run([]string{"--pack", pack, "--output", first}, &stdout, &stderr) == nil {
		t.Fatal("overwrite accepted")
	}
	after, err := os.ReadFile(filepath.Join(first, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("overwrite attempt changed evidence")
	}
}

func TestInvalidCommandDoesNotStartSession(t *testing.T) {
	for _, extra := range [][]string{{"--start", "bad-time"}, {"--start", "2026-10-23T12:00:00.000001Z"}, {"--duration", "1s"}, {"--buffer-window", "0s"}} {
		out := filepath.Join(t.TempDir(), "session")
		args := append([]string{"--pack", "../../../vehicle-packs/alfa-giulia-qv-2017", "--output", out}, extra...)
		if run(args, &bytes.Buffer{}, &bytes.Buffer{}) == nil {
			t.Fatalf("invalid flags accepted: %v", extra)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatal("invalid config created a session")
		}
	}
}
