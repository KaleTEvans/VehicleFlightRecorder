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
if len(archives) < 19:
    raise SystemExit('Expected native baseline, zero-sample and diagnostic no-data/recovery ZIP exports')
expected = {'session.json', 'observations.jsonl', 'events.jsonl', 'incidents.jsonl', 'raw/transport.jsonl'}
scenarios = set()
results = []
live_count = 0
zero_workloads = set()
diagnostic_cases = set()
handoff_cases = set()
can29_cases = set()
regular_workloads = set()
regular_cases = set()
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
    if metadata['schema_version'] == 2 and metadata['live']['workload'] == 'adapter_handoff':
        report=json.loads((destination/'capability.json').read_text())
        if summary['observation_count'] or summary['incident_count'] or report['protocol_profile']!='adapter-state-only' or report['configuration']['selected_protocol']!='unmodified-adapter-state':
            raise SystemExit('Handoff must not claim vehicle acquisition')
        if not report['transactions'] or any(t['key'] not in {'init_previous_protocol','init_previous_description','init_previous_baud'} for t in report['transactions']):
            raise SystemExit('Unexpected handoff command')
        first=report['transactions'][0]
        if first['outcome']=='timeout': handoff_cases.add('timeout')
        elif first['adapter_status']==['34']: handoff_cases.add('retained')
        elif first['adapter_status']==['AUTO']: handoff_cases.add('unknown')
        else: raise SystemExit('Missing native handoff fixture coverage')
    elif metadata['schema_version'] == 2 and metadata['live']['protocol_profile']=='can29-normal-fixed-single-frame-headers-pci' and metadata['live']['workload'] in {'combined','rpm_only','coolant_only','voltage_only'}:
        report=json.loads((destination/'capability.json').read_text())
        workload=metadata['live']['workload']
        regular_workloads.add(workload)
        if report['configuration']['initialization_version']!='obdlink-stp34-runtime-1' or any(t['key']=='can_status' for t in report['transactions']):
            raise SystemExit('Ordinary CAN29 must use its declared runtime version and regular schedules')
        if workload=='combined':
            if summary['incident_count']:
                incident=summary['incidents'][0]
                if summary['incident_count']!=1 or not incident['manual']: raise SystemExit('Expected ordinary manual incident')
                if incident['status']=='captured' and report['generation']==2: regular_cases.add('full_reconnect')
                elif incident['status']=='partial' and report['generation']==1: regular_cases.add('partial')
                else: raise SystemExit('Invalid ordinary manual/reconnect coverage')
                reconstruct(destination,incident['incident_id'])
            elif set(summary['signal_coverage'])=={'electrical.adapter_voltage'}: regular_cases.add('no_data')
            else: raise SystemExit('Missing ordinary no-data case')
        else:
            expected_signals={'rpm_only':{'powertrain.engine.rpm'},'coolant_only':{'powertrain.coolant.temperature'},'voltage_only':{'electrical.adapter_voltage'}}[workload]
            if summary['incident_count'] or set(summary['signal_coverage'])!=expected_signals: raise SystemExit('Invalid individual CAN29 workload')
    elif metadata['schema_version'] == 2 and metadata['live']['workload'] == 'can29_diagnostic':
        report=json.loads((destination/'capability.json').read_text())
        if summary['incident_count'] or metadata['profile_id']!='generic-obd-can29-500' or report['configuration']['initialization_version']!='obdlink-stp34-diagnostic-1':
            raise SystemExit('Invalid native CAN29 profile identity')
        signals=set(summary['signal_coverage'])
        if signals=={'electrical.adapter_voltage'}:
            can29_cases.add('no_data')
        elif signals=={'electrical.adapter_voltage','powertrain.engine.rpm','powertrain.coolant.temperature'}:
            if not {'18DAF110','18DAF118'} <= {c['ecu'] for c in report['capabilities']}:
                raise SystemExit('Missing full CAN29 responder IDs')
            can29_cases.add('timeout' if summary['live']['capability_report']['outcomes'].get('timeout',0) else 'recovery')
        else: raise SystemExit('Invalid CAN29 native signal coverage')
    elif metadata['schema_version'] == 2 and metadata['live']['workload'] == 'discovery_diagnostic':
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
if live_count < 17 or regular_workloads != {'combined','rpm_only','coolant_only','voltage_only'} or regular_cases != {'full_reconnect','partial','no_data'} or can29_cases != {'no_data','recovery','timeout'} or handoff_cases != {'retained','unknown','timeout'} or diagnostic_cases != {'no_data','recovery'} or not {'rpm_only', 'coolant_only'} <= zero_workloads or not {'healthy_drive', 'voltage_tcm_dropout'} <= scenarios:
    raise SystemExit('Native test scenario coverage is incomplete')
(root / 'python-validation.json').write_text(json.dumps(results, indent=2) + '\n')
print(f'Validated {len(archives)} native ZIP exports, including ordinary CAN29 full/partial manual, reconnect, individual/no-data and diagnostic/handoff evidence; reconstructed all available incidents')
