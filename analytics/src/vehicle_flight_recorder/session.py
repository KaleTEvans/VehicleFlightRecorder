"""Strict, streaming inspection of schema-1 Go session exports."""

from collections import Counter
import json
import math
from pathlib import Path


class ValidationError(ValueError):
    """Evidence is malformed or unsupported; no files are modified."""


class IncompleteSessionError(ValidationError):
    """Well-formed preserved records belong to an unfinalized session."""


def _fail(message):
    raise ValidationError(message)


def _object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            _fail(f"duplicate JSON key {key!r}")
        result[key] = value
    return result


def _decode(raw, location):
    try:
        return json.loads(
            raw.decode("utf-8"),
            object_pairs_hook=_object,
            parse_constant=lambda value: _fail(f"non-finite JSON number {value}"),
        )
    except (ValueError, UnicodeError) as error:
        raise ValidationError(f"{location}: {error}") from error


def _integer(value, label, minimum=0):
    if type(value) is not int or value < minimum or value > 2**63 - 1:
        _fail(f"{label}: expected integer in [{minimum}, {2**63 - 1}]")
    return value


def _string(value, label):
    if not isinstance(value, str) or not value.strip():
        _fail(f"{label}: nonempty string required")
    return value


def _schema(record, location):
    if not isinstance(record, dict):
        _fail(f"{location}: JSON object required")
    if type(record.get("schema_version")) is not int or record["schema_version"] != 1:
        _fail(f"{location}: unsupported schema_version {record.get('schema_version')!r}")


def _metadata(metadata):
    _schema(metadata, "session.json")
    for key in ("session_id", "source", "profile_id", "vehicle_pack_version"):
        _string(metadata.get(key), f"session.json {key}")
    start = _integer(metadata.get("start_time_unix_ms"), "session start", 1)
    for key in ("observation_count", "event_count", "incident_count"):
        _integer(metadata.get(key), f"session.json {key}")
    if metadata["incident_count"] != 0:
        _fail("incident records are not supported in Milestone 1")
    if "end_time_unix_ms" not in metadata:
        _fail("session.json end_time_unix_ms is required (null while recording)")
    state = metadata.get("state")
    if state == "recording":
        if metadata["end_time_unix_ms"] is not None:
            _fail("recording session must have null end_time_unix_ms")
    elif state == "complete":
        _integer(metadata["end_time_unix_ms"], "session end", start)
    else:
        _fail(f"unsupported session state {state!r}")
    config = metadata.get("mock")
    if metadata["source"] == "mock" and not isinstance(config, dict):
        _fail("mock source requires reproducibility metadata")
    if config is not None:
        if not isinstance(config, dict) or config.get("scenario") != "healthy_drive" or config.get("generator_version") != "1":
            _fail("unsupported mock scenario/generator_version")
        seed = config.get("seed")
        if type(seed) is not int or not -(2**63) <= seed < 2**63:
            _fail("mock seed must be a signed 64-bit integer")
        duration = _integer(config.get("duration_ms"), "mock duration_ms", 120000)
        if duration > 86400000 or duration % 1000:
            _fail("mock duration must be whole seconds between 2 minutes and 24 hours")
        if type(config.get("step_ms")) is not int or config["step_ms"] != 100:
            _fail("unsupported mock step_ms (expected 100)")
        if start > 2**63 - 1 - duration:
            _fail("mock end time overflows signed 64-bit timestamps")
        if state == "complete" and metadata["end_time_unix_ms"] > start + duration:
            _fail("complete session exceeds configured mock duration")


def _records(path):
    with path.open("rb") as source:
        for line, raw in enumerate(source, 1):
            location = f"{path.name}:{line}"
            if not raw.endswith(b"\n"):
                _fail(f"{location}: truncated JSONL line (missing final newline); evidence preserved")
            yield location, _decode(raw, location)


def _envelope(record, metadata, location, previous):
    _schema(record, location)
    timestamp = _integer(record.get("timestamp_unix_ms"), f"{location} timestamp", 1)
    if timestamp < previous or timestamp < metadata["start_time_unix_ms"]:
        _fail(f"{location}: timestamps must be nondecreasing and at/after session start")
    end = metadata["end_time_unix_ms"]
    if end is not None and timestamp > end:
        _fail(f"{location}: timestamp exceeds session end")
    config = metadata.get("mock")
    if config is not None and timestamp > metadata["start_time_unix_ms"] + config["duration_ms"]:
        _fail(f"{location}: timestamp exceeds configured mock duration")
    for key in ("session_id", "source"):
        if record.get(key) != metadata[key]:
            _fail(f"{location}: {key} does not match session")
    return timestamp


def load_session(directory, *, allow_incomplete=False):
    """Validate an export and summarize without retaining sample arrays.

    Incomplete well-formed logs require explicit opt-in. Finalized counts are
    authoritative only for complete metadata; inspection always counts real rows.
    """
    directory = Path(directory)
    required = ("session.json", "observations.jsonl", "events.jsonl", "incidents.jsonl", "raw/transport.jsonl")
    try:
        for name in required:
            if not (directory / name).is_file():
                _fail(f"missing required file {name}; possible incomplete initialization")
        metadata = _decode((directory / "session.json").read_bytes(), "session.json")
        _metadata(metadata)
        for name in ("incidents.jsonl", "raw/transport.jsonl"):
            if (directory / name).stat().st_size:
                _fail(f"{name}: nonempty records are unsupported in Milestone 1; refusing to ignore evidence")
        coverage = {}
        observations = 0
        last_observation = last_event = metadata["start_time_unix_ms"]
        for location, observation in _records(directory / "observations.jsonl"):
            last_observation = _envelope(observation, metadata, location, last_observation)
            signal = _string(observation.get("signal_id"), f"{location} signal_id")
            unit = _string(observation.get("unit"), f"{location} unit")
            value = observation.get("value")
            try:
                finite = type(value) in (float, int) and math.isfinite(value)
            except OverflowError:
                finite = False
            if not finite:
                _fail(f"{location}: finite numeric value required")
            if observation.get("quality") not in ("valid", "stale", "invalid"):
                _fail(f"{location}: unsupported quality")
            if "vehicle_pack_version" in observation and observation["vehicle_pack_version"] != metadata["vehicle_pack_version"]:
                _fail(f"{location}: vehicle_pack_version mismatch")
            item = coverage.setdefault(signal, {"count": 0, "unit": unit, "first_timestamp_unix_ms": last_observation, "last_timestamp_unix_ms": last_observation})
            if item["unit"] != unit:
                _fail(f"{location}: unit changed for {signal}")
            item["count"] += 1
            item["last_timestamp_unix_ms"] = last_observation
            observations += 1
        events = Counter()
        for location, event in _records(directory / "events.jsonl"):
            last_event = _envelope(event, metadata, location, last_event)
            kind = _string(event.get("event_type"), f"{location} event_type")
            if event.get("severity") not in ("info", "warning", "error"):
                _fail(f"{location}: unsupported severity")
            if "attributes" in event and not isinstance(event["attributes"], dict):
                _fail(f"{location}: attributes must be a JSON object")
            events[kind] += 1
        event_count = sum(events.values())
        if metadata["state"] == "complete":
            if metadata["observation_count"] != observations or metadata["event_count"] != event_count:
                _fail("finalized metadata counts do not match actual JSONL records")
        elif not allow_incomplete:
            raise IncompleteSessionError("session is incomplete (state=recording); use --allow-incomplete to inspect well-formed preserved records; no repair is performed")
        return {
            "schema_version": 1,
            "session_id": metadata["session_id"],
            "source": metadata["source"],
            "profile_id": metadata["profile_id"],
            "vehicle_pack_version": metadata["vehicle_pack_version"],
            "state": metadata["state"],
            "status": "valid" if metadata["state"] == "complete" else "incomplete",
            "start_time_unix_ms": metadata["start_time_unix_ms"],
            "end_time_unix_ms": metadata["end_time_unix_ms"],
            "duration_seconds": (metadata["end_time_unix_ms"] - metadata["start_time_unix_ms"]) / 1000 if metadata["end_time_unix_ms"] is not None else None,
            "observed_duration_seconds": (max(last_observation, last_event) - metadata["start_time_unix_ms"]) / 1000,
            "observation_count": observations,
            "event_count": event_count,
            "incident_count": 0,
            "signal_coverage": dict(sorted(coverage.items())),
            "events_by_type": dict(sorted(events.items())),
        }
    except OSError as error:
        raise ValidationError(f"cannot read session {directory}: {error}") from error
