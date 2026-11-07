"""Real Go live-schema exports; no hardware or physical result is simulated as fact."""
import json
import os
from pathlib import Path
import shutil
import subprocess

import pytest
from vehicle_flight_recorder.session import load_session, ValidationError, IncompleteSessionError
from vehicle_flight_recorder.incidents import reconstruct


@pytest.fixture(scope='module')
def live_exports(tmp_path_factory):
    root = Path(__file__).resolve().parents[2]
    output = tmp_path_factory.mktemp('live')
    binary = output/'live-fixture'
    env = os.environ | {'GOCACHE': str(root/'.cache/go-build'), 'GOMODCACHE': str(root/'.cache/go-mod')}
    subprocess.run(['go','build','-o',str(binary),'./cmd/live-fixture'], cwd=root/'core', env=env, check=True)
    exports = {}
    for scenario in ('healthy','reconnect','malformed','unsupported','timeout'):
        run = subprocess.run([str(binary),'--output',str(output/scenario),'--scenario',scenario], check=True, capture_output=True, text=True)
        exports[scenario] = Path(run.stdout.strip())
    return exports


@pytest.mark.parametrize('scenario', ['healthy','reconnect','malformed','unsupported','timeout'])
def test_live_replay_and_manual_reconstruction(live_exports, scenario):
    result = load_session(live_exports[scenario])
    report = result['live']['capability_report']
    assert result['schema_version'] == 2 and result['source'] == 'obd-live'
    assert result['live']['origin'] == 'scripted-fixture' and report['physical_validation'] is False
    assert report['report_version'] == 2 and report['observation_count'] == result['observation_count']
    assert result['incident_count'] == 1 and result['incidents'][0]['manual'] is True
    incident = result['incidents'][0]
    if scenario == 'reconnect':
        assert report['generation'] == 2
        assert incident['status'] == 'partial' and incident['coverage_gaps']
    else: assert incident['status'] == 'captured'
    if scenario == 'timeout': assert report['outcomes']['timeout'] == 1
    if scenario == 'malformed': assert report['outcomes']['partial'] > 0
    if scenario == 'unsupported': assert set(result['signal_coverage']) == {'electrical.adapter_voltage'}
    detail = reconstruct(live_exports[scenario], incident['incident_id'])
    assert detail


@pytest.mark.parametrize('mutation', ['raw','raw_missing','value','reference','generation','measurement','canonical_missing','schema','origin','unknown','initialization','schedule'])
def test_live_corruption_rejected(live_exports, tmp_path, mutation):
    path = tmp_path/'session'
    shutil.copytree(live_exports['healthy'],path)
    if mutation in ('raw','raw_missing'):
        file = path/'raw/transport.jsonl'; rows = file.read_text().splitlines()
        if mutation == 'raw':
            row = json.loads(rows[0]); row['bytes_base64'] = 'QVRaDQ=='; rows[0] = json.dumps(row)
        else: rows.pop()
        file.write_text('\n'.join(rows)+'\n')
    elif mutation in ('value','reference','generation','measurement','canonical_missing'):
        file = path/'observations.jsonl'; rows = file.read_text().splitlines(); row = json.loads(rows[0])
        if mutation == 'value': row['value'] += 1
        if mutation == 'reference': row['provenance']['raw_refs'] = [1]
        if mutation == 'generation': row['provenance']['generation'] += 1
        if mutation == 'measurement': row['provenance']['measurement_unix_ms'] = row['timestamp_unix_ms']
        rows[0] = json.dumps(row)
        if mutation == 'canonical_missing': rows.pop()
        file.write_text('\n'.join(rows)+'\n')
    elif mutation in ('schema','origin','unknown'):
        file = path/'session.json'; row = json.loads(file.read_text())
        if mutation == 'schema': row['schema_version'] = 1
        if mutation == 'origin': row['live']['origin'] = 'physical'
        if mutation == 'unknown': row['enhanced'] = True
        file.write_text(json.dumps(row))
    else:
        file = path/'capability.json'; row = json.loads(file.read_text())
        if mutation == 'initialization': row['transactions'][14]['adapter_status'] = ['34']
        else: row['schedule'][-1]['interval_ns'] = 1
        file.write_text(json.dumps(row))
    with pytest.raises(ValidationError): load_session(path)


def test_interrupted_live_prefix_requires_opt_in(live_exports, tmp_path):
    path = tmp_path/'session'; shutil.copytree(live_exports['healthy'],path)
    file = path/'session.json'; metadata = json.loads(file.read_text())
    metadata['state'] = 'recording'; metadata['end_time_unix_ms'] = None
    metadata['observation_count'] = metadata['event_count'] = metadata['incident_count'] = 0
    file.write_text(json.dumps(metadata))
    # A report checkpoint may precede the last canonical write at interruption.
    file = path/'observations.jsonl'; rows = file.read_text().splitlines(); removed = json.loads(rows.pop()); file.write_text('\n'.join(rows)+'\n')
    events = path/'events.jsonl'
    prefix = [row for row in events.read_text().splitlines() if json.loads(row)['record_sequence'] < removed['record_sequence']]
    events.write_text('\n'.join(prefix)+'\n')
    with pytest.raises(IncompleteSessionError): load_session(path)
    result = load_session(path, allow_incomplete=True)
    assert result['status'] == 'incomplete'
    assert result['live']['uncommitted_report_observations'] == 1


@pytest.fixture(scope='module')
def diagnostic_exports(tmp_path_factory):
    root = Path(__file__).resolve().parents[2]
    output = tmp_path_factory.mktemp('diagnostic')
    binary = output/'live-fixture'
    env = os.environ | {'GOCACHE': str(root/'.cache/go-build'), 'GOMODCACHE': str(root/'.cache/go-mod')}
    subprocess.run(['go','build','-o',str(binary),'./cmd/live-fixture'], cwd=root/'core', env=env, check=True)
    result = {}
    for scenario in ('no_data','recovery'):
        run = subprocess.run([str(binary),'--output',str(output/scenario),'--workload','discovery_diagnostic','--scenario',scenario], check=True, capture_output=True, text=True)
        result[scenario] = Path(run.stdout.strip())
    return result


@pytest.mark.parametrize('scenario', ['no_data','recovery'])
def test_diagnostic_raw_replay_preserves_unknown_and_recovery(diagnostic_exports, scenario):
    result = load_session(diagnostic_exports[scenario])
    assert result['live']['capability_report']['status'] == 'valid'
    assert result['incident_count'] == 0
    report = json.loads((diagnostic_exports[scenario]/'capability.json').read_text())
    assert report['configuration']['initialization_version'] == 'obdlink-stp33-diagnostic-2'
    assert len([tx for tx in report['transactions'] if tx['key']=='support']) >= 3
    assert any(tx['key']=='can_status' and tx['outcome']=='ok' for tx in report['transactions'])
    if scenario == 'no_data':
        assert set(result['signal_coverage']) == {'electrical.adapter_voltage'}
        assert report['capabilities'] and all(c['ecu']=='adapter' and c['key']=='adapter_voltage' for c in report['capabilities'])
    else:
        assert set(result['signal_coverage']) == {'electrical.adapter_voltage','powertrain.engine.rpm','powertrain.coolant.temperature'}


@pytest.mark.parametrize('mutation', ['version','baud','status','workload'])
def test_diagnostic_profile_or_status_corruption_rejected(diagnostic_exports, tmp_path, mutation):
    path = tmp_path/'session'
    shutil.copytree(diagnostic_exports['no_data'], path)
    report_path=path/'capability.json'
    report=json.loads(report_path.read_text())
    if mutation=='version': report['configuration']['initialization_version']='obdlink-stp33-runtime-1'
    if mutation=='workload': report['configuration']['workload']='combined'
    if mutation=='baud': next(tx for tx in report['transactions'] if tx['key']=='init_baud_check')['adapter_status']=['250000']
    if mutation=='status': next(tx for tx in report['transactions'] if tx['key']=='can_status')['adapter_status']=['T:OFF R:FF']
    report_path.write_text(json.dumps(report))
    with pytest.raises(ValidationError): load_session(path)


@pytest.fixture(scope='module')
def handoff_exports(tmp_path_factory):
    root=Path(__file__).resolve().parents[2]
    output=tmp_path_factory.mktemp('handoff')
    binary=output/'live-fixture'
    env=os.environ|{'GOCACHE':str(root/'.cache/go-build'),'GOMODCACHE':str(root/'.cache/go-mod')}
    subprocess.run(['go','build','-o',str(binary),'./cmd/live-fixture'],cwd=root/'core',env=env,check=True)
    result={}
    for scenario in ('handoff_retained','handoff_unknown','handoff_timeout'):
        run=subprocess.run([str(binary),'--output',str(output/scenario),'--workload','adapter_handoff','--scenario',scenario],check=True,capture_output=True,text=True)
        result[scenario]=Path(run.stdout.strip())
    return result


@pytest.mark.parametrize('scenario',['handoff_retained','handoff_unknown','handoff_timeout'])
def test_adapter_only_handoff_replay(handoff_exports,scenario):
    result=load_session(handoff_exports[scenario])
    assert result['state']=='complete' and result['observation_count']==0 and result['incident_count']==0
    report=json.loads((handoff_exports[scenario]/'capability.json').read_text())
    assert report['protocol_profile']=='adapter-state-only'
    assert report['configuration']['selected_protocol']=='unmodified-adapter-state'
    assert set(t['key'] for t in report['transactions']) <= {'init_previous_protocol','init_previous_description','init_previous_baud'}
    assert not report['observations'] and not report['capabilities']
    if scenario=='handoff_timeout':assert report['transactions'][0]['outcome']=='timeout'
    else:assert len(report['transactions'])==3 and report['transactions'][0]['adapter_status']==(['34'] if scenario=='handoff_retained' else ['AUTO'])


@pytest.mark.parametrize('mutation',['profile','version','readback','scope'])
def test_handoff_does_not_invent_vehicle_configuration(handoff_exports,tmp_path,mutation):
    path=tmp_path/'session';shutil.copytree(handoff_exports['handoff_retained'],path)
    report_path=path/'capability.json';report=json.loads(report_path.read_text())
    if mutation=='profile':report['configuration']['selected_protocol']='ISO15765-4-CAN11-500'
    if mutation=='version':report['configuration']['initialization_version']='obdlink-stp33-diagnostic-2'
    if mutation=='readback':report['transactions'][0]['adapter_status']=['33']
    if mutation=='scope':report['protocol_profile']='can11-single-frame-headers-pci'
    report_path.write_text(json.dumps(report))
    with pytest.raises(ValidationError):load_session(path)

@pytest.fixture(scope='module')
def can29_exports(tmp_path_factory):
    root=Path(__file__).resolve().parents[2]
    output=tmp_path_factory.mktemp('can29')
    binary=output/'live-fixture'
    env=os.environ|{'GOCACHE':str(root/'.cache/go-build'),'GOMODCACHE':str(root/'.cache/go-mod')}
    subprocess.run(['go','build','-o',str(binary),'./cmd/live-fixture'],cwd=root/'core',env=env,check=True)
    result={}
    for scenario in ('healthy','no_data','recovery','malformed','timeout','reconnect','unsupported'):
        run=subprocess.run([str(binary),'--output',str(output/scenario),'--workload','can29_diagnostic','--scenario',scenario],check=True,capture_output=True,text=True)
        result[scenario]=Path(run.stdout.strip())
    return result


@pytest.mark.parametrize('scenario',['healthy','no_data','recovery','malformed','timeout','reconnect','unsupported'])
def test_can29_profile_independent_replay(can29_exports,scenario):
    result=load_session(can29_exports[scenario])
    report=json.loads((can29_exports[scenario]/'capability.json').read_text())
    assert result['profile_id']=='generic-obd-can29-500'
    assert report['protocol_profile']=='can29-normal-fixed-single-frame-headers-pci'
    assert report['configuration']['initialization_version']=='obdlink-stp34-diagnostic-1'
    assert report['configuration']['selected_protocol']=='ISO15765-4-CAN29-500'
    assert not result['incidents']
    assert all(o['ecu']=='adapter' or (len(o['ecu'])==8 and o['ecu'].startswith('18DAF1')) for o in report['observations'])
    if scenario in ('no_data','unsupported'):
        assert set(result['signal_coverage'])=={'electrical.adapter_voltage'}
    else:
        assert set(result['signal_coverage'])=={'electrical.adapter_voltage','powertrain.engine.rpm','powertrain.coolant.temperature'}
        assert {'18DAF110','18DAF118'} <= {c['ecu'] for c in report['capabilities']}
    if scenario=='timeout': assert result['live']['capability_report']['outcomes']['timeout']>=1
    if scenario=='reconnect': assert report['generation']==2
    if scenario=='malformed': assert result['live']['capability_report']['outcomes']['partial']>=1


@pytest.mark.parametrize('mutation',['metadata','catalog','protocol','initialization','header','readback','destination'])
def test_can29_scope_and_raw_corruption_rejected(can29_exports,tmp_path,mutation):
    import base64
    path=tmp_path/'session';shutil.copytree(can29_exports['healthy'],path)
    if mutation=='metadata':
        p=path/'session.json';data=json.loads(p.read_text());data['profile_id']='generic-obd-can11-500';p.write_text(json.dumps(data))
    else:
        p=path/'capability.json';data=json.loads(p.read_text())
        if mutation=='catalog': data['catalog_version']='standard-live-can11-1'
        if mutation=='protocol': data['configuration']['selected_protocol']='ISO15765-4-CAN11-500'
        if mutation=='initialization': data['configuration']['initialization_version']='obdlink-stp33-diagnostic-2'
        if mutation=='readback': next(t for t in data['transactions'] if t['key']=='init29_protocol_check')['adapter_status']=['33']
        if mutation in ('header','destination'):
            # Change both copies of the ledger, forcing independent command or
            # responder validation rather than only the ledger-equality check.
            for row in data['raw']:
                raw=base64.b64decode(row['bytes_base64'])
                raw=raw.replace(b'ATSH18DB33F1',b'ATSH000007DF') if mutation=='header' else raw.replace(b'18DAF110',b'18DA1010')
                row['bytes_base64']=base64.b64encode(raw).decode()
            (path/'raw/transport.jsonl').write_text(''.join(json.dumps(r)+'\n' for r in data['raw']))
        p.write_text(json.dumps(data))
    with pytest.raises(ValidationError):load_session(path)


def test_independent_can29_parser_address_and_duplicate_isolation():
    from vehicle_flight_recorder.acquisition import replay_values,support_bits
    assert replay_values('rpm',b'18 DA F1 10 04 41 0C 1A F8\r>',True)=={'18DAF110':1726}
    assert replay_values('rpm',b'7E8 04 41 0C 1A F8\r>',True)=={}
    assert replay_values('rpm',b'18DAF110 04 41 0C 1A F8\r>',False)=={}
    for address in ('18DA10F1','18DB33F1','18DAF1F1','38DAF110'):
        assert replay_values('rpm',f'{address} 04 41 0C 1A F8\r>'.encode(),True)=={}
    assert replay_values('rpm',b'18DAF110 04 41 0C 00 00\r18DAF110 broken\r18DAF118 04 41 0C 0F A0\r>',True)=={'18DAF118':1000}
    assert support_bits(b'18DAF110 06 41 00 08 10 00 00\r18DAF110 broken\r>',True)=={}

@pytest.fixture(scope='module')
def regular_can29_exports(tmp_path_factory):
    root=Path(__file__).resolve().parents[2]
    output=tmp_path_factory.mktemp('regular-can29')
    binary=output/'live-fixture'
    env=os.environ|{'GOCACHE':str(root/'.cache/go-build'),'GOMODCACHE':str(root/'.cache/go-mod')}
    subprocess.run(['go','build','-o',str(binary),'./cmd/live-fixture'],cwd=root/'core',env=env,check=True)
    cases=[('combined','healthy'),('rpm_only','healthy'),('coolant_only','healthy'),('voltage_only','healthy'),('combined','reconnect'),('combined','no_data'),('combined','timeout')]
    exports={}
    for workload,scenario in cases:
        run=subprocess.run([str(binary),'--output',str(output/f'{workload}-{scenario}'),'--profile','can29','--workload',workload,'--scenario',scenario],check=True,capture_output=True,text=True)
        exports[workload,scenario]=Path(run.stdout.strip())
    return exports


@pytest.mark.parametrize('workload,scenario',[('combined','healthy'),('rpm_only','healthy'),('coolant_only','healthy'),('voltage_only','healthy'),('combined','reconnect'),('combined','no_data'),('combined','timeout')])
def test_regular_can29_manual_replay(regular_can29_exports,workload,scenario):
    path=regular_can29_exports[workload,scenario]
    result=load_session(path)
    report=json.loads((path/'capability.json').read_text())
    assert result['profile_id']=='generic-obd-can29-500'
    assert report['configuration']['initialization_version']=='obdlink-stp34-runtime-1'
    assert report['protocol_profile']=='can29-normal-fixed-single-frame-headers-pci'
    assert not any(t['key']=='can_status' for t in report['transactions'])
    expected={'combined':{'powertrain.engine.rpm','powertrain.coolant.temperature','electrical.adapter_voltage'},'rpm_only':{'powertrain.engine.rpm'},'coolant_only':{'powertrain.coolant.temperature'},'voltage_only':{'electrical.adapter_voltage'}}[workload]
    if scenario=='no_data':expected={'electrical.adapter_voltage'}
    assert set(result['signal_coverage'])==expected
    incident=result['incidents'][0]
    assert incident['manual'] is True
    assert incident['status']==('partial' if scenario=='reconnect' else 'captured')
    assert reconstruct(path,incident['incident_id'])
    if scenario=='reconnect': assert report['generation']==2
    if scenario=='timeout': assert result['live']['capability_report']['outcomes']['timeout']>=1
    if workload=='voltage_only': assert not any(t['key'] in {'support','rpm','coolant'} for t in report['transactions'])


@pytest.mark.parametrize('mutation',['initialization','diagnostic_schedule','metadata_profile','metadata_envelope','workload','selected_protocol'])
def test_regular_can29_contract_rejects_crossed_configuration(regular_can29_exports,tmp_path,mutation):
    path=tmp_path/'session';shutil.copytree(regular_can29_exports['combined','healthy'],path)
    report_path=path/'capability.json';report=json.loads(report_path.read_text())
    if mutation=='initialization':report['configuration']['initialization_version']='obdlink-stp34-diagnostic-1'
    if mutation=='diagnostic_schedule':next(s for s in report['schedule'] if s['key']=='support')['interval_ns']=15*10**9
    if mutation=='selected_protocol':report['configuration']['selected_protocol']='ISO15765-4-CAN11-500'
    if mutation in ('metadata_profile','metadata_envelope','workload'):
        p=path/'session.json';metadata=json.loads(p.read_text())
        if mutation=='metadata_profile':metadata['live']['protocol_profile']='can11-single-frame-headers-pci'
        elif mutation=='metadata_envelope':
            metadata['profile_id']='generic-obd-can11-500'
            metadata['live']['protocol_profile']='can11-single-frame-headers-pci'
            metadata['live']['catalog_version']='standard-live-can11-1'
        else:metadata['live']['workload']='can29_diagnostic';report['configuration']['workload']='can29_diagnostic'
        p.write_text(json.dumps(metadata))
    report_path.write_text(json.dumps(report))
    with pytest.raises(ValidationError):load_session(path)
