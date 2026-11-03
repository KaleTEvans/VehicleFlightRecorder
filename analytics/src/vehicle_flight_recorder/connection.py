"""Validate version-1 transport-only captures independently of drive sessions."""
import base64
import binascii
from pathlib import Path
from .session import ValidationError, _decode


def validate_connection(path, *, allow_incomplete=False):
    path = Path(path)
    if path.stat().st_size > 64 * 1024 * 1024:
        raise ValidationError("connection capture exceeds 64 MiB")
    identity = None
    anchor_ns = anchor_ms = previous_ns = None
    sequence = received = accepted = requested = 0
    pending = b""
    closed = False
    allowed = {"opened", "streams_ready", "received", "write_requested", "write_accepted", "foreground", "background", "inactive", "error", "identity_timeout", "closed"}
    fields = {"capture_version", "connection_id", "sequence", "kind", "native_monotonic_ns", "mapped_unix_ms", "bytes_base64", "detail", "metadata"}
    try:
        with path.open("rb") as source:
            while line := source.readline(128 * 1024 + 1):
                if len(line) > 128 * 1024 or not line.endswith(b"\n"):
                    raise ValidationError("oversized or truncated connection row")
                row = _decode(line, f"connection row {sequence + 1}")
                if not isinstance(row, dict) or set(row) - fields:
                    raise ValidationError("unknown connection row fields")
                for key in ["capture_version", "sequence", "native_monotonic_ns", "mapped_unix_ms"]:
                    if type(row.get(key)) is not int or not 0 <= row[key] <= 2**63 - 1:
                        raise ValidationError(f"invalid {key}")
                if row["capture_version"] != 1 or row["sequence"] != sequence + 1 or closed:
                    raise ValidationError("unsupported version, sequence gap or data after close")
                kind = row.get("kind")
                if kind not in allowed:
                    raise ValidationError("unknown connection event")
                ns, ms = row["native_monotonic_ns"], row["mapped_unix_ms"]
                if sequence == 0:
                    identity = row.get("connection_id")
                    if not isinstance(identity, str) or len(identity) != 43 or not identity.startswith("connection-"):
                        raise ValidationError("invalid connection identity")
                    int(identity[11:], 16)
                    if kind != "opened" or not isinstance(row.get("metadata"), dict) or ns < 0 or ms <= 0:
                        raise ValidationError("missing connection anchor/metadata")
                    anchor_ns, anchor_ms, previous_ns = ns, ms, ns
                elif row.get("connection_id") != identity or kind == "opened" or "metadata" in row:
                    raise ValidationError("foreign connection or repeated metadata")
                if ns < previous_ns or ms != anchor_ms + (ns - anchor_ns) // 1_000_000:
                    raise ValidationError("invalid monotonic mapping")
                previous_ns = ns
                if not isinstance(row.get("detail", ""), str) or len(row.get("detail", "").encode()) > 1024:
                    raise ValidationError("invalid detail")
                encoded = row.get("bytes_base64", "")
                if not isinstance(encoded, str):
                    raise ValidationError("invalid byte encoding")
                data = base64.b64decode(encoded, validate=True)
                if len(data) > 65536:
                    raise ValidationError("oversized chunk")
                if kind in {"received", "write_requested", "write_accepted"}:
                    if not data:
                        raise ValidationError("empty transport bytes")
                elif data:
                    raise ValidationError("unexpected bytes on lifecycle event")
                if kind == "write_requested":
                    if requested or data != b"STI\r":
                        raise ValidationError("unsupported or repeated command")
                    pending = data
                    requested = len(data)
                elif kind == "write_accepted":
                    if not pending.startswith(data):
                        raise ValidationError("accepted bytes differ from pending command")
                    pending = pending[len(data):]
                    accepted += len(data)
                elif kind == "received":
                    received += len(data)
                closed = kind == "closed"
                sequence += 1
    except (ValueError, TypeError, KeyError, binascii.Error, UnicodeError) as error:
        raise ValidationError(f"invalid connection capture: {error}") from error
    if not sequence or (not closed and not allow_incomplete):
        raise ValidationError("empty or interrupted connection capture; use --allow-incomplete to inspect")
    return {"capture_version": 1, "connection_id": identity, "state": "closed" if closed else "interrupted", "record_count": sequence,
            "received_bytes": received, "requested_bytes": requested, "stream_accepted_bytes": accepted,
            "unsent_bytes": len(pending), "adapter_delivery": "unproven", "drive_session": False}
