"""Independent replay of the bounded read-only CAN fault snapshot."""

import base64
import re

FAULT_PREFIXES = {
    "monitor_status": b"\x41\x01",
    "confirmed_codes": b"\x43",
    "pending_codes": b"\x47",
    "permanent_codes": b"\x4a",
    "freeze_frame_dtc": b"\x42\x02\x00",
    "freeze_frame_rpm": b"\x42\x0c\x00",
    "freeze_frame_coolant": b"\x42\x05\x00",
}
FAULT_COMMANDS = {key: bytes([prefix[0]-64, *prefix[1:]]).hex().upper().encode() + b"\r"
                  for key, prefix in FAULT_PREFIXES.items()}
CODE_QUERIES = {"confirmed_codes", "pending_codes", "permanent_codes"}


def fault_responses(key, frame, can29):
    # Import after the acquisition module loads, where this helper is called.
    from .acquisition import response_id

    if not frame.endswith(b">") or frame.count(b">") != 1 or len(frame) > 8192:
        return []
    chunks = {}
    for line in frame[:-1].decode("ascii").replace("\n", "\r").split("\r"):
        compact = "".join(line.upper().split())
        ecu = response_id(compact, can29)
        if ecu is not None:
            chunks.setdefault(ecu, []).append(compact[len(ecu):])
    results = []
    for ecu, lines in chunks.items():
        if not all(re.fullmatch(r"(?:[0-9A-F]{2}){2,8}", line) for line in lines):
            continue
        frames = [bytes.fromhex(line) for line in lines]
        head, *tail = frames
        if 1 <= head[0] <= min(7, len(head)-1) and not tail:
            payload = head[1:1+head[0]]
        elif head[0] >> 4 == 1 and len(head) == 8:
            size = (head[0] & 15) * 256 + head[1]
            if not 8 <= size <= 512 or len(tail) != (size-6+6)//7:
                continue
            if any(row[0] != 0x20 | (index % 16) or
                   len(row)-1 < min(7, size-6-7*(index-1))
                   for index, row in enumerate(tail, 1)):
                continue
            payload = (head[2:] + b"".join(row[1:] for row in tail))[:size]
        else:
            continue
        result = _decode(key, ecu, payload)
        if result is not None:
            results.append(result)
    return results


def _code(a, b):
    return f"{'PCBU'[a >> 6]}{a >> 4 & 3:X}{a & 15:X}{b:02X}"


def _decode(key, ecu, payload):
    prefix = FAULT_PREFIXES[key]
    result = {"ecu": ecu, "outcome": "reply"}
    if len(payload) == 3 and payload[:2] == bytes([0x7F, prefix[0]-64]):
        values = payload[2:]
        result["outcome"] = "negative_response"
    elif payload.startswith(prefix):
        values = payload[len(prefix):]
        if key == "monitor_status":
            if len(values) != 4:
                return None
            result.update(mil=bool(values[0] & 128), code_count=values[0] & 127)
        elif key in CODE_QUERIES:
            if not values or len(values) != 1 + 2*values[0]:
                return None
            pairs = list(zip(values[1::2], values[2::2]))
            if (0, 0) in pairs:
                return None
            if pairs:
                result["codes"] = [_code(a, b) for a, b in pairs]
            else:
                result["outcome"] = "no_codes"
        else:
            if len(values) != (1 if key == "freeze_frame_coolant" else 2):
                return None
            if key == "freeze_frame_dtc":
                if values == b"\0\0":
                    result["outcome"] = "no_freeze_frame_dtc"
                else:
                    result["codes"] = [_code(*values)]
    else:
        return None
    result["data_base64"] = base64.b64encode(values).decode()
    return result
