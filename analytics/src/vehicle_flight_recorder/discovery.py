"""Inventory standard Mode 01 support and measured cadence from a VFR capture."""

import base64
from collections import Counter, defaultdict
from pathlib import Path

from .acquisition import MEASUREMENT_PIDS, SIGNALS, SUPPORT_BLOCKS, support_bits
from .session import ValidationError, _decode, load_session


def inventory(directory, *, allow_incomplete=False):
    directory = Path(directory)
    summary = load_session(directory, allow_incomplete=allow_incomplete)
    if not summary.get("live"):
        raise ValidationError("PID inventory requires a live OBD recording")
    report = _decode((directory / "capability.json").read_bytes(), "capability.json")
    can29 = report["protocol_profile"] == "can29-normal-fixed-single-frame-headers-pci"

    frames = defaultdict(bytearray)
    for row in report["raw"]:
        if row["kind"] == "received":
            frames[row["transaction_id"]].extend(base64.b64decode(row["bytes_base64"]))

    masks = {}
    for tx in report["transactions"]:
        block = SUPPORT_BLOCKS.get(tx["key"])
        if block is not None and tx["outcome"] in {"ok", "partial"}:
            for ecu, mask in support_bits(frames[tx["id"]], can29, block).items():
                masks[tx["generation"], ecu, block] = mask

    # Retain the address and connection boundary. A bitmap from one responder
    # or generation cannot establish another responder's support.
    advertised = defaultdict(list)
    next_blocks = []
    for (generation, ecu, block), mask in sorted(masks.items()):
        for offset in range(1, 32):
            if mask & (1 << (32 - offset)):
                advertised[block + offset].append({"generation": generation, "ecu": ecu})
        if block < 0xE0 and mask & 1 and (generation, ecu, block + 0x20) not in masks:
            next_blocks.append({"generation": generation, "ecu": ecu,
                                "block": f"{block + 0x20:02X}"})

    keys_by_pid = {pid: key for key, (pid, _) in MEASUREMENT_PIDS.items()}
    tx_by_key = defaultdict(list)
    for tx in report["transactions"]:
        tx_by_key[tx["key"]].append(tx)
    observations = defaultdict(list)
    for observation in report["observations"]:
        observations[observation["signal_id"], observation["generation"], observation["ecu"]].append(
            observation["receive_ns"]
        )
    series = defaultdict(list)
    for (signal, generation, ecu), times in sorted(observations.items()):
        times.sort()
        gaps = [later - earlier for earlier, later in zip(times, times[1:])]
        series[signal].append({
            "generation": generation,
            "ecu": ecu,
            "samples": len(times),
            "first_receipt_seconds": times[0] / 1e9,
            "last_receipt_seconds": times[-1] / 1e9,
            "longest_between_samples_seconds": max(gaps) / 1e9 if gaps else None,
        })

    pids = sorted(set(advertised) | {pid for pid, key in keys_by_pid.items() if tx_by_key[key]})
    rows = []
    for pid in pids:
        key = keys_by_pid.get(pid)
        signal = SIGNALS[key][0] if key else None
        transactions = tx_by_key.get(key, ())
        rows.append({
            "pid": f"{pid:02X}",
            "advertised_by": advertised[pid],
            "signal": signal,
            "requests": len(transactions),
            "outcomes": dict(sorted(Counter(tx["outcome"] for tx in transactions).items())),
            "observed": series[signal],
        })
    return {
        "session_id": summary["session_id"],
        "duration_seconds": report["duration_ns"] / 1e9,
        "unread_continuation_blocks": next_blocks,
        "pids": rows,
        "adapter_voltage": series[SIGNALS["adapter_voltage"][0]],
    }
