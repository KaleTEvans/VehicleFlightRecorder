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
