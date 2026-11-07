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


def _schema(record, location, version=1):
    if not isinstance(record, dict):
        _fail(f"{location}: JSON object required")
    if type(record.get("schema_version")) is not int or record["schema_version"] != version:
        _fail(f"{location}: unsupported schema_version {record.get('schema_version')!r}")


def _metadata(metadata):
    version = metadata.get('schema_version') if isinstance(metadata, dict) else None
    if type(version) is not int or version not in (1, 2): _fail('unsupported session schema_version')
    _schema(metadata, "session.json", version)
    if version == 2:
        expected = {'schema_version','session_id','source','profile_id','vehicle_pack_version','start_time_unix_ms','end_time_unix_ms','state','observation_count','event_count','incident_count','incident_capture','live'}
        if set(metadata) != expected or metadata['source'] != 'obd-live' or metadata['vehicle_pack_version'] != 'none': _fail('unknown/incompatible schema_version 2 live metadata')
        live = metadata['live']
        if not isinstance(live, dict) or set(live) != {'origin','operating_state','workload','native_anchor_ns','catalog_version','protocol_profile','report_file','start_origin'}: _fail('invalid live configuration')
        if live['origin'] not in ('physical','native-fixture','scripted-fixture') or live['operating_state'] not in ('unknown','ignition_on_engine_off','stationary_running') or live['workload'] not in ('combined','rpm_only','coolant_only','voltage_only','discovery_diagnostic','can29_diagnostic','adapter_handoff'): _fail('unknown live source/configuration')
        _integer(live['native_anchor_ns'], 'native anchor')
        handoff=live['workload']=='adapter_handoff'
        can29=live['protocol_profile']=='can29-normal-fixed-single-frame-headers-pci'
        regular=live['workload'] in ('combined','rpm_only','coolant_only','voltage_only')
        if not regular and can29 != (live['workload']=='can29_diagnostic'): _fail('live workload/protocol mismatch')
        if handoff and (metadata['observation_count'] != 0 or metadata['incident_count'] != 0): _fail('adapter handoff cannot contain measurements/incidents')
        if metadata['profile_id'] != ('generic-obd-can29-500' if can29 else 'adapter-state-only' if handoff else 'generic-obd-can11-500'): _fail('live workload/profile mismatch')
        if (live['catalog_version'], live['protocol_profile'], live['report_file'], live['start_origin']) != (('standard-live-can29-1','can29-normal-fixed-single-frame-headers-pci','capability.json','manual') if can29 else ('adapter-state-1','adapter-state-only','capability.json','manual') if handoff else ('standard-live-can11-1','can11-single-frame-headers-pci','capability.json','manual')): _fail('unsupported live catalog/profile')
    elif 'live' in metadata or metadata.get('source') == 'obd-live': _fail('live metadata requires schema 2')
    for key in ("session_id", "source", "profile_id", "vehicle_pack_version"):
        _string(metadata.get(key), f"session.json {key}")
    start = _integer(metadata.get("start_time_unix_ms"), "session start", 1)
    for key in ("observation_count", "event_count", "incident_count"):
        _integer(metadata.get(key), f"session.json {key}")
    capture = metadata.get("incident_capture")
    if capture is not None:
        if not isinstance(capture, dict):
            _fail("incident_capture must be an object")
        for key in ("pre_window_ms", "post_window_ms", "max_silence_ms"):
            value = _integer(capture.get(key), key, 1 if key == "max_silence_ms" else 0)
            if value > 86400000:
                _fail("capture window exceeds 24 hours")
    if metadata["incident_count"] and capture is None:
        _fail("incident count requires capture configuration")
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
        if not isinstance(config, dict) or config.get("scenario") not in ("healthy_drive", "voltage_tcm_dropout", "tcm_dropout", "mechanical_fault", "adapter_disconnect", "manual_mark") or config.get("generator_version") not in ("1", "2") or (config.get("scenario") != "healthy_drive" and config.get("generator_version") != "2"):
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
        total = line = 0
        while raw := source.readline(1024*1024+1):
            line += 1; total += len(raw)
            location = f"{path.name}:{line}"
            if len(raw) > 1024*1024 or total > 64*1024*1024: _fail(f'{location}: evidence size budget')
            if not raw.endswith(b"\n"):
                _fail(f"{location}: truncated JSONL line (missing final newline); evidence preserved")
            yield location, _decode(raw, location)


def _envelope(record, metadata, location, previous):
    _schema(record, location, metadata["schema_version"])
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
        with (directory / 'session.json').open('rb') as f: data = f.read(1024*1024+1)
        if len(data) > 1024*1024: _fail('session metadata size budget')
        metadata = _decode(data, "session.json")
        _metadata(metadata)
        live = None
        if metadata['schema_version'] == 2:
            from .live import LiveEvidence
            live = LiveEvidence(directory, metadata)
        elif (directory/'raw/transport.jsonl').stat().st_size:
            _fail('raw/transport.jsonl: nonempty records are unsupported; refusing to ignore evidence')
        coverage = {}
        observations = 0
        last_observation = last_event = metadata["start_time_unix_ms"]
        for location, observation in _records(directory / "observations.jsonl"):
            last_observation = _envelope(observation, metadata, location, last_observation)
            if live: live.observation(observation)
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
        from .incidents import inspect_incidents
        incidents = inspect_incidents(directory, metadata)
        if metadata["state"] == "complete":
            if metadata["observation_count"] != observations or metadata["event_count"] != event_count:
                _fail("finalized metadata counts do not match actual JSONL records")
        elif not allow_incomplete:
            raise IncompleteSessionError("session is incomplete (state=recording); use --allow-incomplete to inspect well-formed preserved records; no repair is performed")
        live_result = live.finish() if live else None
        return {
            **({"live": live_result} if live_result else {}),
            "schema_version": metadata["schema_version"],
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
            "incident_count": len(incidents),
            "trigger_count": sum(len(i["triggers"]) for i in incidents),
            "incidents": incidents,
            "signal_coverage": dict(sorted(coverage.items())),
            "events_by_type": dict(sorted(events.items())),
        }
    except OSError as error:
        raise ValidationError(f"cannot read session {directory}: {error}") from error
