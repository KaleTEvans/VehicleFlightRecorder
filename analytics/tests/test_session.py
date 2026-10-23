import json

import pytest

from vehicle_flight_recorder.__main__ import main
from vehicle_flight_recorder.session import IncompleteSessionError, ValidationError, load_session


def update_metadata(directory, **updates):
    path = directory / "session.json"
    data = json.loads(path.read_text()) | updates
    path.write_text(json.dumps(data) + "\n")


def update_observation(directory, **updates):
    path = directory / "observations.jsonl"
    rows = [json.loads(line) for line in path.read_text().splitlines()]
    rows[0].update(updates)
    path.write_text("".join(json.dumps(row) + "\n" for row in rows))


def test_valid_export_summary(session_dir):
    summary = load_session(session_dir)
    assert summary["status"] == "valid"
    assert summary["duration_seconds"] == 120
    assert summary["observation_count"] == 2
    assert summary["event_count"] == 2
    assert summary["incident_count"] == 0
    assert summary["signal_coverage"]["powertrain.engine.rpm"]["unit"] == "rpm"
    assert load_session(session_dir) == summary


@pytest.mark.parametrize("updates,match", [
    ({"schema_version": 2}, "schema_version"),
    ({"schema_version": True}, "schema_version"),
    ({"observation_count": 3}, "counts"),
    ({"observation_count": True}, "integer"),
    ({"incident_count": 1}, "incident"),
    ({"end_time_unix_ms": None}, "session end"),
    ({"end_time_unix_ms": 100}, "session end"),
    ({"profile_id": ""}, "profile_id"),
    ({"mock": None}, "reproducibility"),
])
def test_bad_metadata(session_dir, updates, match):
    update_metadata(session_dir, **updates)
    with pytest.raises(ValidationError, match=match):
        load_session(session_dir)


@pytest.mark.parametrize("updates,match", [
    ({"schema_version": 2}, "schema_version"),
    ({"timestamp_unix_ms": 999}, "timestamp"),
    ({"timestamp_unix_ms": 122000}, "exceeds"),
    ({"session_id": "foreign"}, "session_id"),
    ({"source": "real"}, "source"),
    ({"unit": ""}, "unit"),
    ({"value": None}, "numeric"),
    ({"value": True}, "numeric"),
    ({"value": float("nan")}, "non-finite"),
    ({"value": 10**1000}, "numeric"),
    ({"quality": "unknown"}, "quality"),
    ({"vehicle_pack_version": "wrong"}, "pack_version"),
])
def test_bad_observation(session_dir, updates, match):
    update_observation(session_dir, **updates)
    with pytest.raises(ValidationError, match=match):
        load_session(session_dir)


def test_timestamp_order(session_dir):
    path = session_dir / "observations.jsonl"
    path.write_text("\n".join(reversed(path.read_text().splitlines())) + "\n")
    with pytest.raises(ValidationError, match="nondecreasing"):
        load_session(session_dir)


def test_missing_observation_value(session_dir):
    path = session_dir / "observations.jsonl"
    rows = [json.loads(line) for line in path.read_text().splitlines()]
    del rows[0]["value"]
    path.write_text("".join(json.dumps(row) + "\n" for row in rows))
    with pytest.raises(ValidationError, match="numeric"):
        load_session(session_dir)


@pytest.mark.parametrize("updates,match", [
    ({"schema_version": 2}, "schema_version"),
    ({"severity": "guess"}, "severity"),
    ({"attributes": []}, "attributes"),
    ({"event_type": ""}, "event_type"),
    ({"source": "physical"}, "source"),
])
def test_bad_event(session_dir, updates, match):
    path = session_dir / "events.jsonl"
    rows = [json.loads(line) for line in path.read_text().splitlines()]
    rows[0].update(updates)
    path.write_text("".join(json.dumps(row) + "\n" for row in rows))
    with pytest.raises(ValidationError, match=match):
        load_session(session_dir)


def test_incomplete_truncation_is_invalid_not_silently_recovered(session_dir, capsys):
    update_metadata(session_dir, state="recording", end_time_unix_ms=None)
    path = session_dir / "events.jsonl"
    path.write_text(path.read_text() + '{"event_type":')
    assert main(["validate", str(session_dir), "--allow-incomplete"]) == 1
    assert "events.jsonl:3" in capsys.readouterr().err


def test_explicit_early_stop_preserves_planned_duration(session_dir):
    update_metadata(session_dir, end_time_unix_ms=1000)
    for name in ("observations.jsonl", "events.jsonl"):
        path = session_dir / name
        rows = [json.loads(line) | {"timestamp_unix_ms": 1000} for line in path.read_text().splitlines()]
        path.write_text("".join(json.dumps(row) + "\n" for row in rows))
    summary = load_session(session_dir)
    assert summary["status"] == "valid" and summary["duration_seconds"] == 0


@pytest.mark.parametrize("name", ["observations.jsonl", "events.jsonl"])
@pytest.mark.parametrize("tail", ['{"schema_version":', '{}', '\n'])
def test_corrupt_line_is_not_discarded(session_dir, name, tail):
    path = session_dir / name
    path.write_text(path.read_text() + tail)
    before = path.read_bytes()
    with pytest.raises(ValidationError, match=name):
        load_session(session_dir, allow_incomplete=True)
    assert path.read_bytes() == before


def test_missing_file_and_future_evidence(session_dir):
    raw = session_dir / "raw/transport.jsonl"
    raw.write_text('{}\n')
    with pytest.raises(ValidationError, match="unsupported"):
        load_session(session_dir)
    raw.unlink()
    with pytest.raises(ValidationError, match="missing required"):
        load_session(session_dir)


def test_invalid_json_and_duplicate_keys(session_dir):
    path = session_dir / "session.json"
    path.write_text('{"schema_version":1,"schema_version":2}\n')
    with pytest.raises(ValidationError, match="duplicate"):
        load_session(session_dir)


def test_incomplete_inspection_without_repair(session_dir, capsys):
    update_metadata(session_dir, state="recording", end_time_unix_ms=None, observation_count=0, event_count=0)
    before = {p.relative_to(session_dir): p.read_bytes() for p in session_dir.rglob("*") if p.is_file()}
    with pytest.raises(IncompleteSessionError):
        load_session(session_dir)
    assert main(["validate", str(session_dir)]) == 2
    assert "incomplete session" in capsys.readouterr().err
    assert main(["summarize", str(session_dir), "--allow-incomplete"]) == 0
    summary = json.loads(capsys.readouterr().out)
    assert summary["status"] == "incomplete" and summary["duration_seconds"] is None
    assert summary["observation_count"] == 2
    assert {p.relative_to(session_dir): p.read_bytes() for p in session_dir.rglob("*") if p.is_file()} == before


def test_cli_exit_status_and_summary(session_dir, capsys):
    assert main(["validate", str(session_dir)]) == 0
    assert json.loads(capsys.readouterr().out)["status"] == "valid"
    assert main(["summarize", str(session_dir)]) == 0
    assert json.loads(capsys.readouterr().out)["signal_coverage"]
    update_metadata(session_dir, schema_version=99)
    assert main(["validate", str(session_dir)]) == 1
    captured = capsys.readouterr()
    assert "schema_version" in captured.err and not captured.out
