import base64
import json
import pytest
from vehicle_flight_recorder.connection import validate_connection
from vehicle_flight_recorder.session import ValidationError


def rows():
    result = []
    for kind, data in [("opened", None), ("streams_ready", None), ("write_requested", b"STI\r"), ("write_accepted", b"S"), ("write_accepted", b"TI\r"), ("received", b"STI\rSTN"), ("received", b"2232 v5.10.3\r"), ("received", b">"), ("closed", None)]:
        row = dict(capture_version=1, connection_id="connection-" + "a" * 32, sequence=len(result)+1, kind=kind, native_monotonic_ns=100+len(result), mapped_unix_ms=1000)
        if kind == "opened": row["metadata"] = {"source": "fixture"}
        if data: row["bytes_base64"] = base64.b64encode(data).decode()
        result.append(row)
    return result


def save(tmp_path, records):
    path = tmp_path / "connection.jsonl"
    path.write_text("".join(json.dumps(row)+"\n" for row in records))
    return path


def test_transcript(tmp_path):
    summary = validate_connection(save(tmp_path, rows()))
    assert summary["stream_accepted_bytes"] == 4
    assert summary["received_bytes"] == 21
    assert summary["drive_session"] is False


@pytest.mark.parametrize("change", ["version", "sequence", "foreign", "command", "accepted", "clock", "encoding", "unknown", "after_close"])
def test_invalid(tmp_path, change):
    records = rows()
    if change == "version": records[0]["capture_version"] = 2
    if change == "sequence": records[1]["sequence"] = 9
    if change == "foreign": records[1]["connection_id"] = "connection-" + "b"*32
    if change == "command": records[2]["bytes_base64"] = base64.b64encode(b"ATZ\r").decode()
    if change == "accepted": records[3]["bytes_base64"] = base64.b64encode(b"X").decode()
    if change == "clock": records[1]["native_monotonic_ns"] = 0
    if change == "encoding": records[5]["bytes_base64"] = "!!"
    if change == "unknown": records[0]["unknown"] = True
    if change == "after_close": records.append(records[-1] | {"sequence": 10})
    with pytest.raises(ValidationError): validate_connection(save(tmp_path, records))


def test_interrupted_and_partial(tmp_path):
    records = rows()[:4]
    path = save(tmp_path, records)
    with pytest.raises(ValidationError): validate_connection(path)
    summary = validate_connection(path, allow_incomplete=True)
    assert summary["unsent_bytes"] == 3
    assert summary["state"] == "interrupted"
    path.write_bytes(path.read_bytes()[:-1])
    with pytest.raises(ValidationError): validate_connection(path, allow_incomplete=True)
