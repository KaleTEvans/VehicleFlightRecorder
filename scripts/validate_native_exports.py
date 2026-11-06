#!/usr/bin/env python3
"""Validate ZIPs copied from a tested iOS app container, never Linux fixtures."""
import json
from pathlib import Path
import sys
import zipfile

from vehicle_flight_recorder.session import load_session
from vehicle_flight_recorder.incidents import reconstruct

root = Path(sys.argv[1])
archives = sorted((root / 'Exports').glob('*.zip'))
if len(archives) < 7:
    raise SystemExit('Expected native baseline, zero-sample and diagnostic no-data/recovery ZIP exports')
expected = {'session.json', 'observations.jsonl', 'events.jsonl', 'incidents.jsonl', 'raw/transport.jsonl'}
scenarios = set()
results = []
live_count = 0
zero_workloads = set()
diagnostic_cases = set()
for archive in archives:
    destination = root / 'unpacked' / archive.stem
    with zipfile.ZipFile(archive) as source:
        metadata = json.loads(source.read('session.json'))
        layout = expected | ({'capability.json'} if metadata['schema_version'] == 2 else set())
        if set(source.namelist()) != layout or len(source.namelist()) != len(layout) or source.testzip() is not None:
            raise SystemExit(f'Invalid native ZIP layout/integrity: {archive}')
        source.extractall(destination)
    metadata = json.loads((destination / 'session.json').read_text())
    if metadata['schema_version'] == 2:
        if metadata['live']['origin'] != 'native-fixture':
            raise SystemExit('Native public artifact must be explicitly synthetic')
        live_count += 1
    else:
        scenarios.add(metadata['mock']['scenario'])
    summary = load_session(destination)
    if summary['state'] != 'complete':
        raise SystemExit('Native test export must be finalized')
    if metadata['schema_version'] == 2 and metadata['live']['workload'] == 'discovery_diagnostic':
        report = json.loads((destination/'capability.json').read_text())
        if summary['incident_count'] != 0 or report['configuration']['initialization_version'] != 'obdlink-stp33-diagnostic-2':
            raise SystemExit('Invalid native diagnostic fixture identity')
        if len([t for t in report['transactions'] if t['key']=='support']) < 3 or not any(t['key']=='can_status' and t['outcome']=='ok' for t in report['transactions']):
            raise SystemExit('Missing native diagnostic retry/status evidence')
        signals=set(summary['signal_coverage'])
        if signals == {'electrical.adapter_voltage'} and report['capabilities'] and all(c['ecu']=='adapter' and c['key']=='adapter_voltage' for c in report['capabilities']): diagnostic_cases.add('no_data')
        elif signals == {'electrical.adapter_voltage','powertrain.engine.rpm','powertrain.coolant.temperature'}: diagnostic_cases.add('recovery')
        else: raise SystemExit('Invalid native diagnostic coverage')
    elif metadata['schema_version'] == 2 and summary['observation_count'] == 0:
        if summary['incident_count'] != 0 or summary['live']['capability_report']['outcomes'].get('no_data', 0) < 1:
            raise SystemExit('Zero-sample export must preserve its real scripted no-data transaction')
        zero_workloads.add(metadata['live']['workload'])
    else:
        if summary['incident_count'] != 1:
            raise SystemExit('Native manual/fault test export must have one incident')
        incident = summary['incidents'][0]
        if incident['status'] != 'captured':
            raise SystemExit('Native scenario did not capture full evidence')
        reconstruction = reconstruct(destination, incident['incident_id'])
        reconstructed = root / 'reconstructions'
        reconstructed.mkdir(exist_ok=True)
        (reconstructed / f'{archive.stem}.json').write_text(json.dumps(reconstruction, indent=2) + '\n')
    results.append(summary)
if live_count < 5 or diagnostic_cases != {'no_data','recovery'} or not {'rpm_only', 'coolant_only'} <= zero_workloads or not {'healthy_drive', 'voltage_tcm_dropout'} <= scenarios:
    raise SystemExit('Native test scenario coverage is incomplete')
(root / 'python-validation.json').write_text(json.dumps(results, indent=2) + '\n')
print(f'Validated {len(archives)} native ZIP exports, including zero-sample RPM/coolant and diagnostic no-data/recovery evidence; reconstructed all available incidents')
