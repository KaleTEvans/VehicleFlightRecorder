"""Acceptance checks consume real Go CLI output, not the parser unit fixtures."""

import json
import subprocess
import sys

from vehicle_flight_recorder.session import load_session


def test_real_go_export_is_reproducible_and_python_readable(go_command, tmp_path):
    executable, pack = go_command
    directories = [tmp_path / "a", tmp_path / "b"]
    for directory in directories:
        result = subprocess.run([str(executable), "--pack", str(pack), "--output", str(directory), "--duration", "2m", "--seed", "42", "--session-id", "integration"], check=True, capture_output=True, text=True)
        assert json.loads(result.stdout)["state"] == "complete"
    first, second = directories
    for path in first.rglob("*"):
        if path.is_file():
            assert path.read_bytes() == (second / path.relative_to(first)).read_bytes()
    summary = load_session(first)
    assert summary["status"] == "valid"
    assert summary["observation_count"] == 5166
    assert summary["event_count"] == 9
    assert summary["events_by_type"] == {"mock_phase_changed": 7, "recording_started": 1, "recording_stopped": 1}
    assert summary["duration_seconds"] == 120
    assert summary["incident_count"] == 0
    assert len(summary["signal_coverage"]) == 6
    assert summary["signal_coverage"]["powertrain.engine.rpm"]["count"] == 1201
    assert summary["signal_coverage"]["powertrain.coolant.temperature"]["count"] == 121
    for command in ("validate", "summarize"):
        result = subprocess.run([sys.executable, "-m", "vehicle_flight_recorder", command, str(first)], check=True, capture_output=True, text=True)
        assert json.loads(result.stdout)["observation_count"] == 5166
    before = (first / "session.json").read_bytes()
    failed = subprocess.run([str(executable), "--pack", str(pack), "--output", str(first)], capture_output=True, text=True)
    assert failed.returncode != 0
    assert (first / "session.json").read_bytes() == before
