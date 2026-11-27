"""Independent validation of synthetic acquisition reports, never live sessions."""
import base64
from collections import Counter, defaultdict
import math
import json
from pathlib import Path
import re
from .faults import FAULT_COMMANDS, fault_responses
from .session import ValidationError, _decode, _integer

COMMANDS = {"support": b"0100\r", "rpm": b"010C\r", "coolant": b"0105\r", "adapter_voltage": b"STVR\r"}
SIGNALS = {"rpm": ("powertrain.engine.rpm", "rpm"), "coolant": ("powertrain.coolant.temperature", "degC"), "adapter_voltage": ("electrical.adapter_voltage", "V")}
SUPPORT_BLOCKS = {"support": 0, "support20": 0x20, "support40": 0x40}
MEASUREMENT_PIDS = {"rpm": (0x0C, 2), "coolant": (0x05, 1), "speed": (0x0D, 1),
                    "short_trim_bank1": (0x06, 1), "long_trim_bank1": (0x07, 1),
                    "short_trim_bank2": (0x08, 1), "long_trim_bank2": (0x09, 1),
                    "oil_temperature": (0x5C, 1), "fuel_rate": (0x5E, 2)}
DIAGNOSTIC_PIDS = {"calculated_load": (0x04, 1), "manifold_pressure": (0x0B, 1),
                   "intake_temperature": (0x0F, 1), "throttle_position": (0x11, 1),
                   "fuel_rail_pressure": (0x23, 2)}
MEASUREMENT_PIDS.update(DIAGNOSTIC_PIDS)
SIGNALS.update({"calculated_load": ("powertrain.engine.calculated_load", "%"),
                "manifold_pressure": ("powertrain.intake.manifold_pressure", "kPa"),
                "intake_temperature": ("powertrain.intake.temperature", "degC"),
                "throttle_position": ("powertrain.throttle.position", "%"),
                "fuel_rail_pressure": ("powertrain.fuel.rail_pressure_gauge", "kPa")})
FUEL_TRIMS = {key for key in MEASUREMENT_PIDS if "trim_bank" in key}
SIGNALS.update({"speed": ("powertrain.vehicle.speed", "km/h"),
                "oil_temperature": ("powertrain.oil.temperature", "degC"),
                "fuel_rate": ("powertrain.fuel.rate", "L/h")})
for bank in (1, 2):
    for term in ("long", "short"):
        SIGNALS[f"{term}_trim_bank{bank}"] = (f"powertrain.fuel.trim.{term}.bank{bank}", "%")
EXPANDED_COMMANDS = {key: f"01{pid:02X}\r".encode() for key, pid in
                     (SUPPORT_BLOCKS | {k: v[0] for k, v in MEASUREMENT_PIDS.items()}).items()
                     if key not in COMMANDS}
EXPANDED_SCHEDULE = {"support20": (60, 0), "support40": (60, 0), "speed": (2, 4),
                     "oil_temperature": (15, 6), "fuel_rate": (5, 5)}
EXPANDED_SCHEDULE.update({key: (10, 5) for key in FUEL_TRIMS})

DIAGNOSTIC_RECORDING_SCHEDULE = {
    "support20": (60, 0), "speed": (3, 4), "calculated_load": (5, 5),
    "manifold_pressure": (5, 5), "throttle_position": (5, 5), "fuel_rail_pressure": (5, 5),
    "short_trim_bank1": (10, 6), "long_trim_bank1": (15, 6),
    "short_trim_bank2": (10, 6), "long_trim_bank2": (15, 6), "intake_temperature": (15, 7),
}

def advertising_block(key):
    pid = SUPPORT_BLOCKS[key] if key in SUPPORT_BLOCKS else MEASUREMENT_PIDS[key][0]
    return (pid - 1) // 32 * 32, 31 - (pid - 1) % 32

INITIALIZATION = dict(zip(
    ('init_close init_defaults init_echo init_linefeeds init_memory init_spaces init_headers init_dlc init_format init_flow init_responses init_timing init_protocol init_timeout init_protocol_check init_identity').split(),
    (b'STPC\r', b'ATD\r', b'ATE0\r', b'ATL0\r', b'ATM0\r', b'ATS1\r', b'ATH1\r', b'ATD0\r', b'ATCAF1\r', b'ATCFC1\r', b'ATR1\r', b'ATAT0\r', b'STP33\r', b'STPTO500\r', b'STPR\r', b'STI\r')))
DIAGNOSTIC_INITIALIZATION = dict(list(INITIALIZATION.items())[:13] + [
    ('init_functional_header', b'ATSH7DF\r'), ('init_normal_addressing', b'STCAF0\r'),
    ('init_receive_defaults', b'ATCRA\r'), ('init_automatic_filters', b'STFA\r'),
    ('init_baud_check', b'STPBRR\r')] + list(INITIALIZATION.items())[13:])
HANDOFF_INITIALIZATION = {'init_previous_protocol': b'STPR\r', 'init_previous_description': b'STPRS\r', 'init_previous_baud': b'STPBRR\r'}
CAN29_INITIALIZATION = {({'init_protocol':'init29_protocol', 'init_functional_header':'init29_functional_header', 'init_protocol_check':'init29_protocol_check'}.get(k,k)):
    ({'init_protocol':b'STP34\r', 'init_functional_header':b'ATSH18DB33F1\r'}.get(k,v)) for k,v in DIAGNOSTIC_INITIALIZATION.items()}
ALL_INITIALIZATION = DIAGNOSTIC_INITIALIZATION | CAN29_INITIALIZATION | HANDOFF_INITIALIZATION
DISCOVERY_COMMANDS = {f"mode01_{pid:02x}": f"01{pid:02X}\r".encode() for pid in range(256)}
LIVE_COMMANDS = COMMANDS | EXPANDED_COMMANDS | ALL_INITIALIZATION | {'can_status': b'ATCS\r'}


def response_id(compact, can29=False):
    if can29:
        return compact[:8] if re.match(r'18DAF1[0-9A-F]{2}',compact) and compact[6:8] != 'F1' else None
    return compact[:3] if re.match(r'7E[8-F]',compact) else None


def support_bits(data, can29=False, block=0):
    candidates, seen, bad = {}, set(), set()
    for line in data.decode('ascii', errors='replace').replace('\n', '\r').replace('>', '\r').split('\r'):
        compact = ''.join(line.upper().split())
        ecu = response_id(compact,can29)
        if ecu is None: continue
        if not re.fullmatch(r'(?:[0-9A-F]{2}){2,8}', compact[len(ecu):]):
            bad.add(ecu)
            continue
        payload = bytes.fromhex(compact[len(ecu):])
        if ecu in seen: bad.add(ecu)
        seen.add(ecu)
        if payload[0] == 6 and len(payload) >= 7 and payload[1:3] == bytes((0x41, block)):
            candidates[ecu] = int.from_bytes(payload[3:7], 'big')
        else: bad.add(ecu)
    return {ecu: mask for ecu, mask in candidates.items() if ecu not in bad}


def discovery_responders(frame, pid, can29):
    if not frame.endswith(b'>') or frame.count(b'>') != 1 or len(frame) > 8192:
        return {}
    lines = frame[:-1].decode('ascii').replace('\n', '\r').split('\r')
    matched, seen, bad = {}, set(), set()
    for line in lines:
        compact = ''.join(line.upper().split())
        ecu = response_id(compact, can29)
        if ecu is None:
            continue
        if ecu in seen:
            bad.add(ecu)
        seen.add(ecu)
        text = compact[len(ecu):]
        if not re.fullmatch(r"(?:[0-9A-F]{2}){2,8}", text):
            continue
        data = bytes.fromhex(text)
        if 3 <= data[0] <= 7 and data[0] < len(data) and data[1:3] == bytes((0x41, pid)):
            matched[ecu] = data[3:1+data[0]].hex()
    return {ecu: value for ecu, value in matched.items() if ecu not in bad}


def initialization_value(key, raw):
    try:
        require(all(b in (9,10,13) or 32 <= b <= 126 for b in raw), 'initialization ASCII')
        text = raw.strip().decode('ascii')
        require(text.endswith('>') and text.count('>') == 1, 'initialization prompt')
        lines = [x.strip() for x in text[:-1].replace('\n', '\r').split('\r') if x.strip()]
        lines = [x for x in lines if ''.join(x.upper().split()) != ALL_INITIALIZATION[key].decode().strip()]
        require(len(lines) == 1, 'ambiguous initialization reply')
        if key in HANDOFF_INITIALIZATION:
            require(len(lines[0]) <= 128 and all(c == '\t' or 32 <= ord(c) <= 126 for c in lines[0]), 'handoff readback shape')
            return lines[0]
        if key == 'init_identity': require(re.fullmatch(r'STN[0-9]{4} v[0-9]+(?:\.[0-9]+){1,3}', lines[0]), 'firmware identity')
        else: require(lines[0] == ('34' if key == 'init29_protocol_check' else '33' if key == 'init_protocol_check' else '500000' if key == 'init_baud_check' else 'OK'), 'initialization acknowledgement/protocol')
        return lines[0]
    except UnicodeError as error:
        raise ValidationError('invalid initialization ASCII') from error

OUTCOMES = {"pending", "ok", "partial", "no_data", "adapter_error", "negative_response", "unavailable", "protocol_error", "timeout", "cancelled", "resource_limit"}


def require(condition, detail):
    if not condition:
        raise ValidationError("acquisition report: " + detail)


def fields(row, expected):
    require(isinstance(row, dict) and set(row) == set(expected.split()), "unexpected or missing fields")


def number(value):
    require(type(value) in (int, float) and math.isfinite(value), "non-finite or invalid observation")
    return value


def replay_values(key, raw, can29=False):
    """Check the small documented decoder surface against exact received bytes.

    This is an offline integrity check, not an alternative acquisition/parser loop.
    """
    if key in ALL_INITIALIZATION or key == "can_status": return {}
    raw = raw.strip(b" \t\r\n")
    if not raw.endswith(b">") or raw.count(b">") != 1 or len(raw) > 8192:
        return {}
    try:
        lines = raw[:-1].decode("ascii").replace("\n", "\r").split("\r")
    except UnicodeError:
        return {}
    result, duplicates, seen_ecus = {}, set(), set()
    for line in lines:
        line = line.strip()
        compact = "".join(line.upper().split())
        if not line or compact == LIVE_COMMANDS[key].decode().strip():
            continue
        if key == "adapter_voltage":
            if re.fullmatch(r"[0-9]{1,2}\.[0-9]{2}", line) and float(line) <= 65.534:
                if "adapter" in result: duplicates.add("adapter")
                result["adapter"] = float(line)
            continue
        ecu = response_id(compact,can29)
        if ecu is None:
            continue
        if ecu in seen_ecus: duplicates.add(ecu)
        seen_ecus.add(ecu)
        if not re.fullmatch(r"(?:[0-9A-F]{2}){2,8}", compact[len(ecu):]):
            duplicates.add(ecu)
            continue
        data = bytes.fromhex(compact[len(ecu):])
        if not 1 <= data[0] <= 7 or data[0]+1 > len(data):
            duplicates.add(ecu)
            continue
        payload = data[1:1+data[0]]
        if key in SUPPORT_BLOCKS:
            continue
        pid, width = MEASUREMENT_PIDS[key]
        lengths = (3, 4) if key in FUEL_TRIMS else (width + 2,)
        if len(payload) not in lengths or payload[:2] != bytes((65, pid)):
            if not (len(payload) == 3 and payload[:2] == b'\x7f\x01'): duplicates.add(ecu)
            continue
        if ecu in result: duplicates.add(ecu)
        value = payload[2]
        if key in ("coolant", "oil_temperature", "intake_temperature"): value -= 40
        elif key == "rpm": value = int.from_bytes(payload[2:], "big") / 4
        elif key in ("calculated_load", "throttle_position"): value = value * 100 / 255
        elif key == "fuel_rail_pressure": value = int.from_bytes(payload[2:], "big") * 10
        elif key == "fuel_rate": value = int.from_bytes(payload[2:], "big") / 20
        elif key in FUEL_TRIMS: value = (value - 128) * 100 / 128
        result[ecu] = value
    return {ecu: value for ecu, value in result.items() if ecu not in duplicates}


def validate_acquisition(path, *, allow_incomplete=False):
    try:
        return _validate_acquisition(path, allow_incomplete=allow_incomplete)
    except ValidationError:
        raise
    except (ValueError, TypeError, KeyError, IndexError, OverflowError) as error:
        raise ValidationError(f"invalid acquisition report: {error}") from error


def _validate_acquisition(path, *, allow_incomplete=False):
    path = Path(path)
    require(path.stat().st_size <= 16 * 1024 * 1024, "16 MiB absolute report limit")
    with path.open("rb") as source:
        data = source.read(16 * 1024 * 1024 + 1)
    require(len(data) <= 16 * 1024 * 1024, "report grew beyond 16 MiB")
    try:
        report = _decode(data, str(path))
    except ValidationError:
        if len(data) > 8 * 1024 * 1024:
            raise ValidationError('report grew beyond 8 MiB without a valid live envelope')
        raise
    version = report.get('report_version') if isinstance(report, dict) else None
    require(type(version) is int and version in (1, 2), 'unsupported version')
    live = version == 2
    fields({key: value for key, value in report.items() if key != "fault_results"}, "report_version kind source protocol_profile catalog_version anchor_unix_ms duration_ns generation state vehicle_identity reference_comparison schedule raw transactions observations capabilities statistics issues" + (' configuration' if live else ''))
    commands = LIVE_COMMANDS if live else COMMANDS
    handoff = live and isinstance(report.get('configuration'),dict) and report['configuration'].get('workload')=='adapter_handoff'
    can29 = live and report['protocol_profile']=='can29-normal-fixed-single-frame-headers-pci'
    profile,catalog = ('can29-normal-fixed-single-frame-headers-pci','standard-live-can29-1') if can29 else ('adapter-state-only','adapter-state-1') if handoff else ('can11-single-frame-headers-pci','standard-live-can11-1')
    require((report['kind'], report['source'], report['protocol_profile'], report['catalog_version']) ==
            (('live-acquisition', 'obd-live', profile, catalog) if live else
             ('offline-acquisition', 'synthetic', 'can11-single-frame-headers-pci', 'standard-offline-1')), 'unsupported source/profile/catalog')
    require(report['vehicle_identity'] == 'unknown' and report['reference_comparison'] == 'unmeasured', 'unproven vehicle/reference claim')
    if not live: require(len(data) <= 8 * 1024 * 1024, '8 MiB offline report limit')
    if live:
        config = report['configuration']
        fields({k:v for k,v in config.items() if k not in {'start_origin','policy_version','startup_sync_version'}}, 'app_version bridge_version reference_note sensor_refresh_rate enhanced_pack maximum_duration_ns minimum_admission_ns session_id origin operating_state workload native_anchor_ns initialization_version selected_protocol connections')
        require(isinstance(config['session_id'], str) and re.fullmatch(r'live-[0-9a-f]{32}', config['session_id']), 'live session identity')
        require(config['origin'] in ('physical', 'native-fixture', 'scripted-fixture') and config['operating_state'] in ('unknown', 'ignition_on_engine_off', 'stationary_running') and config['workload'] in ('combined', 'rpm_only', 'coolant_only', 'voltage_only', 'standard_expanded', 'standard_pid_discovery', 'standard_diagnostic_recording', 'standard_fault_scan', 'discovery_diagnostic', 'can29_diagnostic', 'adapter_handoff'), 'live configuration')
        diagnostic = config['workload'] in ('discovery_diagnostic','can29_diagnostic')
        regular = config['workload'] in ('combined','rpm_only','coolant_only','voltage_only','standard_expanded','standard_pid_discovery','standard_diagnostic_recording','standard_fault_scan')
        require((regular or (can29 == (config['workload']=='can29_diagnostic'))), 'workload/protocol mismatch')
        init_steps = HANDOFF_INITIALIZATION if handoff else CAN29_INITIALIZATION if can29 else DIAGNOSTIC_INITIALIZATION if diagnostic else INITIALIZATION
        require(config['initialization_version'] == (('obdlink-stp34-diagnostic-1' if diagnostic else 'obdlink-stp34-runtime-1') if can29 else 'obdlink-adapter-handoff-1' if handoff else 'obdlink-stp33-diagnostic-2' if diagnostic else 'obdlink-stp33-runtime-1') and config['selected_protocol'] == ('ISO15765-4-CAN29-500' if can29 else 'unmodified-adapter-state' if handoff else 'ISO15765-4-CAN11-500') and config['app_version'] == '0.1.0' and type(config['bridge_version']) is int and config['bridge_version'] == 1, 'configuration version/protocol')
        automatic = 'start_origin' in config or 'policy_version' in config
        maximum = 15 if handoff else 600
        if automatic:
            require(config.get('policy_version') == 'rpm-activity-1' and config.get('start_origin') in ('automatic','activity_probe') and config['operating_state']=='unknown' and config['workload']==('combined' if config['start_origin']=='automatic' else 'rpm_only'), 'automatic policy/configuration mismatch')
            maximum = 1800 if config['start_origin']=='automatic' else 300
        require(config['sensor_refresh_rate'] == 'unknown' and config['enhanced_pack'] == 'none' and config['maximum_duration_ns'] == maximum*10**9 and config['minimum_admission_ns'] == 5*10**8, 'unsupported live policy')
        if 'startup_sync_version' in config: require(automatic and config['startup_sync_version']=='quiet-start-1','unsupported startup synchronization')
        require(isinstance(config['reference_note'], str) and len(config['reference_note'].encode()) <= 512, 'reference note')
        _integer(config['native_anchor_ns'], 'native anchor')
        require(isinstance(config['connections'], list) and len(config['connections']) <= 16, 'connection budget')
        for expected, connection in enumerate(config['connections'], 1):
            fields(connection, 'generation start_ns metadata')
            require(type(connection['generation']) is int and connection['generation'] == expected, 'connection generation sequence')
            _integer(connection['start_ns'], 'connection start')
            require(isinstance(connection['metadata'], dict) and len(connection['metadata']) <= 16 and connection['metadata'].get('protocol') == 'com.obdlink', 'connection identity')
            require(all(isinstance(k, str) and len(k.encode()) <= 64 and isinstance(v, str) and len(v.encode()) <= 512 for k, v in connection['metadata'].items()), 'connection metadata')
    duration = _integer(report['duration_ns'], 'duration', 0 if live else 1)
    require(duration <= 86400 * 10**9, "duration limit")
    anchor = _integer(report['anchor_unix_ms'], 'anchor', 1)
    generation = _integer(report['generation'], 'generation', 1)
    require(report['state'] in {'ready', 'writing', 'awaiting', 'draining', 'blocked', 'disconnected'}, "unknown state")
    require(allow_incomplete or report['state'] in {'blocked', 'disconnected'}, "unfinished acquisition report")
    expanded = live and config['workload'] == 'standard_expanded'
    diagnostic_recording = live and config['workload'] == 'standard_diagnostic_recording'
    discovery = live and config['workload'] == 'standard_pid_discovery'
    fault_scan = live and config['workload'] == 'standard_fault_scan'
    require(fault_scan or "fault_results" not in report, "unexpected fault scan evidence")
    if fault_scan:
        commands = commands | FAULT_COMMANDS
        require(generation == 1 and len(config['connections']) <= 1, 'fault scan crosses connections')
        require(not report['observations'] and not report['capabilities'] and not report['statistics'], 'fault scan cannot claim live measurements')
    if discovery:
        commands = commands | DISCOVERY_COMMANDS
        require(generation == 1 and len(config['connections']) <= 1, 'discovery pass crosses connections')
        require(not report['observations'] and not report['capabilities'] and not report['statistics'], 'discovery must retain raw replies without claiming measurements')
    for key, maximum in [('schedule', 280 if discovery else 40 if expanded or diagnostic_recording else 32 if live else 16), ('raw', 16384 if live else 4096), ('transactions', 2048 if live else 512), ('observations', 8192 if live else 2048), ('capabilities', 512 if can29 or expanded or diagnostic_recording else 32), ('statistics', 512 if can29 or expanded or diagnostic_recording else 32), ('issues', 1024)]:
        require(isinstance(report[key], list) and len(report[key]) <= maximum, f"invalid {key} budget")
    require(all(isinstance(issue, str) for issue in report['issues']), "invalid issues")
    schedule = {}
    for spec in report['schedule']:
        fields(spec, "key interval_ns timeout_ns priority")
        key = spec['key']
        require(key in commands and key not in schedule, "unknown/duplicate request")
        require(10**8 <= _integer(spec['interval_ns'], 'interval') <= 60 * 10**9 and
                10**8 <= _integer(spec['timeout_ns'], 'timeout') <= 5 * 10**9 and
                0 <= _integer(spec['priority'], 'priority') <= 9, "invalid schedule")
        schedule[key] = spec
    require(schedule, "empty schedule")
    if live:
        require(list(schedule)[:len(init_steps)] == list(init_steps), 'missing/reordered initialization configuration')
        selected = set(schedule)-set(init_steps) if discovery or fault_scan else {'combined': {'support', 'rpm', 'coolant', 'adapter_voltage'}, 'standard_expanded': {'support','rpm','coolant','adapter_voltage'} | set(EXPANDED_SCHEDULE), 'standard_diagnostic_recording': {'support','rpm','coolant','adapter_voltage'} | set(DIAGNOSTIC_RECORDING_SCHEDULE), 'rpm_only': {'support', 'rpm'}, 'coolant_only': {'support', 'coolant'}, 'voltage_only': {'adapter_voltage'}, 'discovery_diagnostic': {'support', 'rpm', 'coolant', 'adapter_voltage', 'can_status'}, 'can29_diagnostic': {'support','rpm','coolant','adapter_voltage','can_status'}, 'adapter_handoff': set()}[config['workload']]
        require(set(schedule)-set(init_steps) == selected, 'workload/configuration mismatch')
        require(all(schedule[k] == dict(key=k, interval_ns=60*10**9, timeout_ns=2*10**9, priority=0) for k in init_steps), 'initialization schedule')
        if discovery or fault_scan:
            one_shot_commands = FAULT_COMMANDS if fault_scan else DISCOVERY_COMMANDS
            require(all(key in one_shot_commands and schedule[key] == dict(
                key=key, interval_ns=60*10**9, timeout_ns=2*10**9, priority=0
            ) for key in selected), 'unsupported discovery schedule')
        for key in () if discovery or fault_scan else selected:
            intervals = {'support': (15 if diagnostic else 60, 0), 'rpm': (2, 1),
                         'coolant': (5, 2), 'adapter_voltage': (5, 3), 'can_status': (15, 4)}
            seconds, priority = (intervals | (DIAGNOSTIC_RECORDING_SCHEDULE if diagnostic_recording else EXPANDED_SCHEDULE))[key]
            require(schedule[key] == dict(key=key, interval_ns=seconds*10**9, timeout_ns=2*10**9, priority=priority), 'unsupported live schedule')
    txs, refs = {}, defaultdict(list)
    previous_end = 0
    previous_generation = 1
    for index, tx in enumerate(report['transactions'], 1):
        fields(tx, "id generation key start_ns sent_ns end_ns outcome accepted_bytes raw_refs adapter_status")
        require(tx['id'] == f'tx-{index:04d}' and tx['key'] in schedule and tx['outcome'] in OUTCOMES, "invalid transaction identity/request/outcome")
        tx_generation = _integer(tx['generation'], 'transaction generation', 1)
        require(previous_generation <= tx_generation <= generation, "out-of-order/future generation")
        previous_generation = tx_generation
        start = _integer(tx['start_ns'], 'start')
        require(previous_end <= start <= duration, "overlapping/out-of-order transaction")
        if tx['end_ns'] is None:
            require(allow_incomplete and tx['outcome'] == 'pending' and index == len(report['transactions']), "missing end")
        else:
            end = _integer(tx['end_ns'], 'end')
            require(start <= end <= duration and tx['outcome'] != 'pending', "invalid transaction end")
            previous_end = end
        if tx['sent_ns'] is not None:
            require(start <= _integer(tx['sent_ns'], 'sent') <= (tx['end_ns'] if tx['end_ns'] is not None else duration), "invalid send time")
        require(type(tx['accepted_bytes']) is int and 0 <= tx['accepted_bytes'] <= len(commands[tx['key']]), "invalid accepted count")
        require(isinstance(tx['raw_refs'], list) and all(type(i) is int for i in tx['raw_refs']), "invalid raw references")
        require(isinstance(tx['adapter_status'], list) and all(isinstance(i, str) for i in tx['adapter_status']), "invalid adapter status")
        txs[tx['id']] = tx
    total = previous_ns = 0
    raw_by_tx = defaultdict(list)
    for sequence, raw in enumerate(report['raw'], 1):
        fields(raw, "sequence generation transaction_id kind monotonic_ns bytes_base64" + (" session_id" if live else ""))
        if live: require(raw["session_id"] == config["session_id"], "foreign raw session")
        require(type(raw['sequence']) is int and raw['sequence'] == sequence, "raw sequence gap")
        ns = _integer(raw['monotonic_ns'], 'raw time')
        require(previous_ns <= ns <= duration, "raw time regression")
        previous_ns = ns
        require(raw['kind'] in {'requested', 'accepted', 'received', 'late_received', 'unsolicited'}, "unknown raw event")
        require(isinstance(raw['bytes_base64'], str), "invalid raw encoding")
        try:
            data = base64.b64decode(raw['bytes_base64'], validate=True)
        except ValueError as error:
            raise ValidationError('invalid raw Base64') from error
        total += len(data)
        require(0 < len(data) <= 65536 and total <= (2 * 1024 * 1024 if live else 512 * 1024), "raw byte budget")
        tx = txs.get(raw['transaction_id'])
        if tx is None:
            require(raw['transaction_id'] == '' and raw['kind'] == 'unsolicited', "foreign raw transaction")
            require(type(raw['generation']) is int and 1 <= raw['generation'] <= generation, "raw generation")
            continue
        require(type(raw['generation']) is int and raw['generation'] == tx['generation'], "foreign raw generation")
        require(ns >= tx['start_ns'], "raw before admission")
        if tx['end_ns'] is not None:
            if raw['kind'] == 'late_received': require(ns >= tx['end_ns'] and tx['outcome'] == 'timeout', "invalid late bytes")
            else: require(ns <= tx['end_ns'], "raw after transaction end")
        refs[tx['id']].append(sequence)
        raw_by_tx[tx['id']].append((raw, data))
    expected_values = {}
    for id, tx in txs.items():
        require(tx['raw_refs'] == refs[id], "broken raw reference list")
        rows = raw_by_tx[id]
        requested = [(row, data) for row, data in rows if row['kind'] == 'requested']
        require(len(requested) == 1 and rows[0][0]['kind'] == 'requested' and requested[0][1] == commands[tx['key']] and requested[0][0]['monotonic_ns'] == tx['start_ns'], "invalid command evidence")
        accepted = b''.join(data for row, data in rows if row['kind'] == 'accepted')
        require(commands[tx['key']].startswith(accepted) and len(accepted) == tx['accepted_bytes'], "accepted bytes differ from request")
        if tx['sent_ns'] is not None:
            accepted_rows = [row for row, _ in rows if row['kind'] == 'accepted']
            require(accepted == commands[tx['key']] and accepted_rows[-1]['monotonic_ns'] == tx['sent_ns'], "send time lacks full acceptance evidence")
        received_rows = [(row, data) for row, data in rows if row['kind'] == 'received']
        expected_values[id] = {} if tx['key'] in DISCOVERY_COMMANDS or tx['key'] in FAULT_COMMANDS else replay_values(tx['key'], b''.join(data for _, data in received_rows), can29)
        if tx['outcome'] in {'ok', 'partial'}:
            require(tx['sent_ns'] is not None and received_rows, "success lacks response provenance")
            require(max(row['sequence'] for row, _ in rows if row['kind'] == 'accepted') < received_rows[0][0]['sequence'], "response predates completed write")
    for tx in report['transactions']:
        if tx['key'] == 'can_status' and tx['outcome'] == 'ok':
            frame = b''.join(data for row, data in raw_by_tx[tx['id']] if row['kind'] == 'received')
            require(len(frame) <= 256 and frame.endswith(b'>') and frame.count(b'>') == 1, 'CAN status frame')
            try: lines = [x.strip() for x in frame[:-1].decode('ascii').replace('\n', '\r').split('\r') if x.strip() and ''.join(x.upper().split()) != 'ATCS']
            except UnicodeError as error: raise ValidationError('CAN status ASCII') from error
            require(len(lines) == 1 and len(lines[0]) <= 80 and all(c in '\t' or 32 <= ord(c) <= 126 for c in lines[0]), 'CAN status shape')
            require(lines[0].upper() not in {'?','OK','NO DATA','ERROR','CAN ERROR','BUS ERROR','STOPPED'} and tx['adapter_status'] == lines, 'CAN status receipt mismatch')
    if live:
        positions = defaultdict(int)
        previous_failed = {}
        stopped_counts = defaultdict(int)
        first_admitted = set()
        support_by_generation = defaultdict(dict)
        next_block = 0
        candidates = set()
        probes = None
        discovery_keys = []
        for tx in report['transactions']:
            gen = tx['generation']
            require(1 <= gen <= len(config['connections']), 'transaction lacks connection provenance')
            require(config['connections'][gen-1]['start_ns'] <= tx['start_ns'], 'transaction precedes connection')
            sync=config.get('startup_sync_version')=='quiet-start-1'
            if sync and gen not in first_admitted:
                require(tx['start_ns']>=config['connections'][gen-1]['start_ns']+10**9,'startup quiet interval skipped')
                residual=[row['monotonic_ns'] for row in report['raw'] if row['generation']==gen and row['kind']=='unsolicited' and config['connections'][gen-1]['start_ns']<=row['monotonic_ns']<=tx['start_ns']]
                require(not residual or tx['start_ns']>=max(residual)+5*10**8,'startup residual quiet interval skipped')
                first_admitted.add(gen)
            if tx['key'] in init_steps:
                if gen in previous_failed:
                    failed=previous_failed.pop(gen)
                    require(sync and failed['key']=='init_close' and failed['outcome']=='protocol_error' and failed['adapter_status']==['STOPPED'] and stopped_counts[gen]<=2 and tx['start_ns']>=failed['end_ns']+10**9,'unbounded/unverified initialization retry')
                position = positions[gen]
                require(position < len(init_steps) and tx['key'] == list(init_steps)[position], 'setup stage skipped or repeated')
                if tx['outcome'] == 'ok':
                    frame = b''.join(data for row, data in raw_by_tx[tx['id']] if row['kind'] == 'received')
                    require(tx['adapter_status'] == [initialization_value(tx['key'], frame)], 'setup receipt mismatch')
                    positions[gen] += 1
                else:
                    previous_failed[gen]=tx
                    if tx['key']=='init_close' and tx['adapter_status']==['STOPPED']:
                        require(sync,'STOPPED retry receipt requires declared startup synchronization')
                        frame=b''.join(data for row,data in raw_by_tx[tx['id']] if row['kind']=='received')
                        require(len(frame)<=256 and frame.endswith(b'>') and frame.count(b'>')==1,'invalid stopped-close frame')
                        try: lines=[x.strip() for x in frame[:-1].decode('ascii').replace('\n','\r').split('\r') if x.strip() and x.strip()!='STPC']
                        except UnicodeError as error: raise ValidationError('stopped-close ASCII') from error
                        require(lines==['STOPPED'] and tx['outcome']=='protocol_error','STOPPED cannot acknowledge setup')
                        stopped_counts[gen]+=1
                        require(stopped_counts[gen]<=3,'startup retry limit')
            else:
                require(positions[gen] == len(init_steps), 'measurement requested before setup verification')
                key = tx['key']
                if fault_scan:
                    discovery_keys.append(key)
                    continue
                if discovery:
                    discovery_keys.append(key)
                    pid = int(key.removeprefix('mode01_'), 16)
                    if next_block is not None:
                        require(pid == next_block, 'discovery did not follow an advertised support block')
                        frame = b''.join(data for row, data in raw_by_tx[tx['id']] if row['kind'] == 'received')
                        masks = support_bits(frame, can29, pid) if tx['outcome'] == 'ok' else {}
                        require(tx['outcome'] != 'ok' or masks, 'discovery support success lacks valid reply')
                        candidates.update(pid + offset for mask in masks.values()
                                          for offset in range(1, 32) if mask & (1 << (32-offset)))
                        next_block = pid + 0x20 if pid < 0xE0 and any(mask & 1 for mask in masks.values()) else None
                    else:
                        if probes is None:
                            probes = iter(sorted(candidates))
                        require(pid == next(probes, None), 'discovery probed an unadvertised or repeated PID')
                        if tx['outcome'] == 'ok':
                            frame = b''.join(data for row, data in raw_by_tx[tx['id']] if row['kind'] == 'received')
                            require(discovery_responders(frame, pid, can29), 'discovery success lacks a matching reply')
                    continue
                if key in MEASUREMENT_PIDS or key in ('support20', 'support40'):
                    block, bit = advertising_block(key)
                    require(any(b == block and mask & (1 << bit) for (b, _), mask in support_by_generation[gen].items()), 'measurement requested without prior same-generation advertised support')
                if key in SUPPORT_BLOCKS and tx['outcome'] in {'ok','partial'}:
                    frame = b''.join(data for row, data in raw_by_tx[tx['id']] if row['kind'] == 'received')
                    block = SUPPORT_BLOCKS[key]
                    support_by_generation[gen].update({(block, ecu): mask for ecu, mask in support_bits(frame, can29, block).items()})
        if fault_scan:
            require(discovery_keys == list(FAULT_COMMANDS)[:len(discovery_keys)] and
                    discovery_keys == list(schedule)[len(init_steps):], 'fault scan queries differ from allowlisted sequence')
            results = []
            for tx in report['transactions']:
                if tx['key'] not in FAULT_COMMANDS or tx['outcome'] not in {'ok','partial','negative_response','unavailable'}:
                    continue
                frame = b''.join(data for row, data in raw_by_tx[tx['id']] if row['kind'] == 'received')
                responders = fault_responses(tx['key'], frame, can29)
                if tx['outcome'] != 'unavailable':
                    require(responders and tx['sent_ns'] is not None, 'fault scan result lacks raw response')
                if responders:
                    require(tx['sent_ns'] is not None, 'fault scan reply precedes command acceptance')
                    results.append({"transaction_id": tx['id'], "responders": responders})
            require(json.dumps(report.get('fault_results', []), sort_keys=True) == json.dumps(results, sort_keys=True), 'fault scan results differ from raw evidence')
        if discovery:
            require(discovery_keys == list(schedule)[len(init_steps):], 'discovery schedule differs from captured requests')
    if handoff:
        require(generation == 1 and len(config['connections']) <= 1 and len(report['transactions']) <= 3, 'handoff must not reconnect or retry')
        require([t['key'] for t in report['transactions']] == list(HANDOFF_INITIALIZATION)[:len(report['transactions'])], 'handoff command sequence')
    if handoff: require(not report['observations'] and not report['capabilities'] and not report['statistics'], 'handoff cannot claim vehicle/voltage measurements')
    groups, observed_pairs = defaultdict(list), set()
    for observation in report['observations']:
        fields(observation, "transaction_id generation ecu signal_id value unit receive_ns mapped_unix_ms measurement_unix_ms decoder_version")
        tx = txs.get(observation['transaction_id'])
        require(tx is not None and tx['key'] in SIGNALS and tx['outcome'] in {'ok', 'partial', 'resource_limit'}, "foreign/unsuccessful observation transaction")
        require(type(observation['generation']) is int and observation['generation'] == tx['generation'], "foreign observation generation")
        require((observation['signal_id'], observation['unit']) == SIGNALS[tx['key']] and observation['decoder_version'] == '1', "decoder/catalog mismatch")
        ecu = observation['ecu']
        require((tx['id'], ecu) not in observed_pairs and ecu in expected_values[tx['id']], "duplicate/unproven observation")
        value = number(observation['value'])
        require(value == expected_values[tx['id']][ecu], "observation differs from raw decoder evidence")
        ns = _integer(observation['receive_ns'], 'receive')
        require(tx['sent_ns'] is not None and tx['sent_ns'] <= ns == tx['end_ns'], "observation timing")
        require(any(row['kind'] == 'received' and row['monotonic_ns'] == ns for row, _ in raw_by_tx[tx['id']]), "missing receive reference")
        require(type(observation['mapped_unix_ms']) is int and observation['mapped_unix_ms'] == anchor + ns//1_000_000 and observation['measurement_unix_ms'] is None, "invalid clock mapping or invented measurement time")
        observed_pairs.add((tx['id'], ecu))
        groups[(tx['key'], ecu)].append((ns, ns-tx['sent_ns']))
    for id, tx in txs.items():
        if tx['outcome'] in {'ok', 'partial'} and tx['key'] in SIGNALS:
            require({ecu for tid, ecu in observed_pairs if tid == id} == set(expected_values[id]), "missing decoded observation")
    stats_by_pair = {}
    for stats in report['statistics']:
        fields(stats, "key ecu count rate_hz maximum_gap_ns latency_min_ns latency_p95_ns")
        pair = (stats['key'], stats['ecu'])
        require(pair not in stats_by_pair and pair in groups, "duplicate/foreign statistics")
        values = groups[pair]
        times = [ns for ns, _ in values]
        require(times == sorted(times), "observation ordering")
        gaps = [times[0], duration-times[-1]] + [b-a for a, b in zip(times, times[1:])]
        latencies = sorted(latency for _, latency in values)
        require(type(stats['count']) is int and stats['count'] == len(values) and
                math.isclose(number(stats['rate_hz']), len(values)*10**9/duration, rel_tol=1e-12) and
                type(stats['maximum_gap_ns']) is int and stats['maximum_gap_ns'] == max(gaps) and
                type(stats['latency_min_ns']) is int and stats['latency_min_ns'] == latencies[0] and
                type(stats['latency_p95_ns']) is int and stats['latency_p95_ns'] == latencies[math.ceil(.95*len(latencies))-1], "statistics do not match observation population")
        stats_by_pair[pair] = stats
    require(set(stats_by_pair) == set(groups), "missing statistics")
    support_masks = {}
    for tx in report['transactions']:
        if tx['generation'] != generation or tx['key'] not in SUPPORT_BLOCKS or tx['outcome'] not in {'ok', 'partial'}:
            continue
        data = b''.join(data for row, data in raw_by_tx[tx['id']] if row['kind'] == 'received')
        block = SUPPORT_BLOCKS[tx['key']]
        support_masks.update({(block, ecu): mask for ecu, mask in support_bits(data, can29, block).items()})
    capability_keys = set()
    for cap in report['capabilities']:
        fields(cap, "ecu key state advertised observed evidence_transaction")
        pair = (cap['ecu'], cap['key'])
        require(pair not in capability_keys and (cap['key'] in {'rpm','coolant','adapter_voltage'} or (expanded and cap['key'] in EXPANDED_SCHEDULE) or (diagnostic_recording and cap['key'] in DIAGNOSTIC_RECORDING_SCHEDULE)) and (cap['ecu'] == 'adapter' or response_id(cap['ecu'],can29) == cap['ecu']), "invalid/duplicate capability")
        require(cap['state'] in {'unknown', 'advertised', 'observed', 'unsupported', 'unavailable'} and type(cap['observed']) is bool and (cap['advertised'] is None or type(cap['advertised']) is bool), "invalid capability state")
        evidence = txs.get(cap['evidence_transaction'])
        require(evidence is not None and evidence['generation'] == generation and (evidence['key'] == cap['key'] or evidence['key'] in SUPPORT_BLOCKS), "capability cache crosses generation/configuration")
        if cap['advertised'] is not None:
            require(cap['key'] in MEASUREMENT_PIDS or cap['key'] in ('support20', 'support40'), "invalid advertised PID")
            block, bit = advertising_block(cap['key'])
            if evidence['key'] in SUPPORT_BLOCKS:
                require(SUPPORT_BLOCKS[evidence['key']] == block, "capability references the wrong support block")
            pair_mask = (block, cap['ecu'])
            require(pair_mask in support_masks, "advertisement lacks raw support evidence")
            require(cap['advertised'] == bool(support_masks[pair_mask] & (1 << bit)), "advertisement differs from support bits")
        if cap['state'] == 'unsupported': require(cap['advertised'] is False and not cap['observed'], "timeout inferred unsupported")
        if cap['state'] == 'advertised': require(cap['advertised'] is True and not cap['observed'], "invalid advertised state")
        if cap['state'] == 'observed': require(cap['observed'], "invented observation")
        if cap['state'] == 'unavailable': require(cap['observed'] or cap['advertised'] is True, "unknown inferred unavailable")
        if cap['observed']: require(any(o['generation'] == generation and o['ecu'] == cap['ecu'] and txs[o['transaction_id']]['key'] == cap['key'] for o in report['observations']), "capability observed without measurement")
        capability_keys.add(pair)
    return {'status': 'valid', 'source': report['source'], 'physical_validation': False, 'report_version': version,
            'generation': generation, 'state': report['state'], 'duration_seconds': duration/10**9,
            'transaction_count': len(txs), 'outcomes': dict(sorted(Counter(tx['outcome'] for tx in txs.values()).items())),
            'observation_count': len(observed_pairs), 'raw_record_count': len(report['raw']), 'statistics': report['statistics']}
