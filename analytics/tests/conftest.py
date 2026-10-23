"""Small synthetic exports for error cases, plus real Go output for integration."""

import json
import os
from pathlib import Path
import shutil
import subprocess

import pytest


@pytest.fixture
def session_dir(tmp_path):
    directory = tmp_path / "session"
    (directory / "raw").mkdir(parents=True)
    metadata = {
        "schema_version": 1, "session_id": "fixture", "source": "mock",
        "profile_id": "synthetic.profile", "vehicle_pack_version": "0.1.0",
        "start_time_unix_ms": 1000, "end_time_unix_ms": 121000,
        "state": "complete", "observation_count": 2, "event_count": 2, "incident_count": 0,
        "mock": {"scenario": "healthy_drive", "seed": 42, "duration_ms": 120000, "step_ms": 100, "generator_version": "1"},
    }
    observation = {"schema_version": 1, "timestamp_unix_ms": 1000, "session_id": "fixture", "source": "mock", "signal_id": "powertrain.engine.rpm", "value": 800.0, "unit": "rpm", "quality": "valid", "vehicle_pack_version": "0.1.0"}
    event = {"schema_version": 1, "timestamp_unix_ms": 1000, "session_id": "fixture", "source": "mock", "event_type": "recording_started", "severity": "info"}
    (directory / "session.json").write_text(json.dumps(metadata) + "\n")
    (directory / "observations.jsonl").write_text(json.dumps(observation) + "\n" + json.dumps(observation | {"timestamp_unix_ms": 121000, "value": 0.0}) + "\n")
    (directory / "events.jsonl").write_text(json.dumps(event) + "\n" + json.dumps(event | {"timestamp_unix_ms": 121000, "event_type": "recording_stopped"}) + "\n")
    (directory / "incidents.jsonl").touch()
    (directory / "raw/transport.jsonl").touch()
    return directory


@pytest.fixture(scope="session")
def go_command(tmp_path_factory):
    root = Path(__file__).resolve().parents[2]
    go = shutil.which("go")
    if go is None:
        pytest.fail("Cross-language tests require Go 1.24+ on PATH; run scripts/test_all.sh after configuring Go.")
    executable = tmp_path_factory.mktemp("go-command") / "mock-session"
    env = os.environ | {"GOCACHE": str(root / ".cache/go-build"), "GOMODCACHE": str(root / ".cache/go-mod")}
    subprocess.run([go, "build", "-o", str(executable), "./cmd/mock-session"], cwd=root / "core", env=env, check=True, capture_output=True, text=True)
    return executable, root / "vehicle-packs/alfa-giulia-qv-2017"
