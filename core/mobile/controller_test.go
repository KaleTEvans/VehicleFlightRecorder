package mobile_test

import (
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"vehicleflightrecorder.local/core/mobile"
)

func controller(t *testing.T) *mobile.Controller {
	t.Helper()
	pack, err := filepath.Abs("../../vehicle-packs/alfa-giulia-qv-2017")
	if err != nil {
		t.Fatal(err)
	}
	c, err := mobile.NewController(pack, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}
func snapshot(t *testing.T, c *mobile.Controller) map[string]any {
	t.Helper()
	text, err := c.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func TestMobileScenarioLifecycle(t *testing.T) {
	for name, triggers := range map[string]int{"healthy_drive": 0, "voltage_tcm_dropout": 2, "tcm_dropout": 1, "mechanical_fault": 1, "adapter_disconnect": 1, "manual_mark": 1} {
		t.Run(name, func(t *testing.T) {
			c := controller(t)
			if err := c.Start(name); err != nil {
				t.Fatal(err)
			}
			if err := c.Start(name); err == nil {
				t.Fatal("double start")
			}
			s := snapshot(t, c)
			id := s["session"].(map[string]any)["session_id"].(string)
			if _, err := c.ExportPath(id); err == nil {
				t.Fatal("exported active session")
			}
			for n := 0; n < 72; n++ {
				if err := c.Advance(10000); err != nil {
					t.Fatal(err)
				}
			}
			s = snapshot(t, c)
			if s["state"] != "complete" {
				t.Fatal(s["state"])
			}
			if s["physical_adapter_state"] != "not_connected" {
				t.Fatal("fabricated physical connection")
			}
			items := s["incidents"].([]any)
			if triggers == 0 {
				if len(items) != 0 {
					t.Fatal(items)
				}
			} else {
				if len(items) != 1 || len(items[0].(map[string]any)["triggers"].([]any)) != triggers {
					t.Fatal(items)
				}
				item := items[0].(map[string]any)
				want := "captured"
				if name == "adapter_disconnect" {
					want = "partial"
				}
				if item["status"] != want {
					t.Fatal(item)
				}
				page, err := c.IncidentJSON(id, item["incident_id"].(string), 0)
				if err != nil {
					t.Fatal(err)
				}
				var detail struct {
					Timeline []any `json:"timeline"`
					Next     int64 `json:"next_offset"`
					More     bool  `json:"has_more"`
				}
				if err := json.Unmarshal([]byte(page), &detail); err != nil {
					t.Fatal(err)
				}
				if len(detail.Timeline) != 200 || detail.Next != 200 || !detail.More {
					t.Fatal(detail)
				}
			}
			if _, err := c.ExportPath(id); err != nil {
				t.Fatal(err)
			}
			if _, err := c.SessionsJSON(); err != nil {
				t.Fatal(err)
			}
			if _, err := c.Mark("after"); err == nil {
				t.Fatal("mark after stop")
			}
			if err := c.Start("healthy_drive"); err != nil {
				t.Fatal(err)
			}
			if err := c.Stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestMobileManualErrorsAndConcurrentSnapshots(t *testing.T) {
	c := controller(t)
	if _, err := c.Mark("inactive"); err == nil {
		t.Fatal("inactive mark")
	}
	if err := c.Start("unknown"); err == nil {
		t.Fatal("invalid scenario")
	}
	if err := c.Start("healthy_drive"); err != nil {
		t.Fatal(err)
	}
	if err := c.Advance(1); err == nil {
		t.Fatal("invalid logical step")
	}
	if _, err := c.ExportPath("../escape"); err == nil {
		t.Fatal("path escape")
	}
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 10; n++ {
				if _, err := c.Snapshot(); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	for n := 0; n < 7; n++ {
		if err := c.Advance(10000); err != nil {
			t.Fatal(err)
		}
	}
	id, err := c.Mark("manual")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	s := snapshot(t, c)
	sid := s["session"].(map[string]any)["session_id"].(string)
	if _, err := c.IncidentJSON(sid, id, 999999); err == nil {
		t.Fatal("invalid offset")
	}
	if _, err := c.IncidentJSON(sid, "missing", 0); err == nil {
		t.Fatal("missing incident")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
