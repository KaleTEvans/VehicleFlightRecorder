"""Incident acceptance consumes exports from the actual Go recorder."""

import json
import subprocess
import sys

import pytest

from vehicle_flight_recorder.incidents import reconstruct
from vehicle_flight_recorder.session import ValidationError, load_session

SCENARIOS = {
    'voltage_tcm_dropout': ['voltage_excursion', 'module_timeout'],
    'tcm_dropout': ['module_timeout'],
    'mechanical_fault': ['new_dtc'],
    'adapter_disconnect': ['transport_disconnect'],
    'manual_mark': ['manual_marker'],
}


def generate(go_command, directory, scenario, *flags):
    executable, pack = go_command
    result = subprocess.run([str(executable), '--pack', str(pack), '--output', str(directory), '--duration', '6m', '--scenario', scenario, '--session-id', scenario, *flags], capture_output=True, text=True)
    assert result.returncode == 0, result.stderr
    return json.loads(result.stdout)


@pytest.mark.parametrize('scenario', SCENARIOS)
def test_deterministic_incidents_and_evidence(go_command, tmp_path, scenario):
    a, b = tmp_path / 'a', tmp_path / 'b'
    for path in (a, b):
        assert generate(go_command, path, scenario)['incident_count'] == 1
    for path in a.rglob('*'):
        if path.is_file():
            assert path.read_bytes() == (b / path.relative_to(a)).read_bytes()
    summary = load_session(a)
    assert summary['incident_count'] == 1
    assert summary['trigger_count'] == len(SCENARIOS[scenario])
    item = summary['incidents'][0]
    assert [t['trigger_type'] for t in item['triggers']] == SCENARIOS[scenario]
    assert item['status'] == ('partial' if scenario == 'adapter_disconnect' else 'captured')
    assert item['window_start_unix_ms'] == item['trigger_time_unix_ms'] - 60000
    assert item['window_end_unix_ms'] == item['triggers'][-1]['timestamp_unix_ms'] + 120000
    result = reconstruct(a, item['incident_id'])
    rows = result['timeline']
    keys = [(r['reference']['timestamp_unix_ms'], r['reference']['record_sequence']) for r in rows]
    assert keys == sorted(keys)
    refs = [row['reference'] for row in rows]
    for trigger in item['triggers']:
        assert trigger['evidence'] in refs
    observations = [r['record'] for r in rows if r['reference']['file'] == 'observations.jsonl']
    events = [r['record'] for r in rows if r['reference']['file'] == 'events.jsonl']
    if scenario == 'voltage_tcm_dropout':
        assert item['triggers'][0]['timestamp_unix_ms'] < item['triggers'][1]['timestamp_unix_ms']
        low = min(o['timestamp_unix_ms'] for o in observations if o['signal_id'] == 'electrical.adapter_voltage' and o['value'] < 12)
        latency = min(o['timestamp_unix_ms'] for o in observations if o['signal_id'] == 'network.tcm_response_latency_ms' and o['value'] > 100)
        timeout = min(e['timestamp_unix_ms'] for e in events if e['event_type'] == 'diagnostic_timeout')
        restored = min(e['timestamp_unix_ms'] for e in events if e['event_type'] == 'response_restored')
        assert low < latency < timeout < restored
    if scenario == 'tcm_dropout':
        assert all(o['value'] >= 13 for o in observations if o['signal_id'] == 'electrical.adapter_voltage')
        assert all(t['category'] == 'communication' for t in item['triggers'])
    if scenario == 'mechanical_fault':
        assert item['triggers'][0]['category'] == 'powertrain'
        assert any(o['signal_id'] == 'engine.boost.actual' and o['value'] == 120 and o['timestamp_unix_ms'] < item['trigger_time_unix_ms'] for o in observations)
        assert sum(e['event_type'] == 'dtc_observed' for e in events) == 2
    if scenario == 'adapter_disconnect':
        assert item['coverage_gaps'][0]['reason'] == 'transport_disconnected'
        gap = item['coverage_gaps'][0]
        assert gap['end_time_unix_ms'] - gap['start_time_unix_ms'] == 2000
        assert not any(gap['start_time_unix_ms'] <= o['timestamp_unix_ms'] < gap['end_time_unix_ms'] for o in observations)
        assert 'module_timeout' not in SCENARIOS[scenario]
    if scenario == 'manual_mark':
        assert item['manual'] and item['triggers'][0]['label'] == 'synthetic manual marker'
    cli = subprocess.run([sys.executable, '-m', 'vehicle_flight_recorder', 'incident', str(a), item['incident_id']], check=True, capture_output=True, text=True)
    assert json.loads(cli.stdout) == result
    with pytest.raises(ValidationError, match='unknown incident'):
        reconstruct(a, 'missing')


@pytest.fixture
def partial(go_command, tmp_path):
    directory = tmp_path / 'partial'
    generate(go_command, directory, 'manual_mark', '--duration', '2m')
    return directory


def rewrite(directory, edit):
    path = directory / 'incidents.jsonl'
    records = [json.loads(line) for line in path.read_text().splitlines()]
    edit(records)
    path.write_text(''.join(json.dumps(item) + '\n' for item in records))


def test_partial_post_capture_and_incomplete_inspection(partial):
    item = load_session(partial)['incidents'][0]
    assert item['status'] == 'partial'
    assert item['incomplete_reasons'] == ['post_window_unavailable']
    meta_path = partial / 'session.json'
    meta = json.loads(meta_path.read_text())
    meta.update(state='recording', end_time_unix_ms=None, observation_count=0, event_count=0, incident_count=0)
    meta_path.write_text(json.dumps(meta))
    summary = load_session(partial, allow_incomplete=True)
    assert summary['status'] == 'incomplete' and summary['incident_count'] == 1


@pytest.mark.parametrize('mutation', ['reference', 'revision', 'immutable', 'status', 'schema', 'missing', 'sequence', 'link'])
def test_reject_corrupt_incident_evidence(partial, mutation):
    if mutation == 'sequence':
        path = partial / 'events.jsonl'
        lines = path.read_text().splitlines()
        record = json.loads(lines[0]); record['record_sequence'] = 100
        lines[0] = json.dumps(record); path.write_text('\n'.join(lines) + '\n')
    else:
        def edit(rows):
            if mutation == 'reference':
                for row in rows:
                    row['triggers'][0]['evidence']['line'] = 1
            elif mutation == 'revision': rows[-1]['revision'] += 1
            elif mutation == 'immutable': rows[-1]['triggers'][0]['label'] = 'changed'
            elif mutation == 'status': rows[-1]['status'] = 'captured'
            elif mutation == 'schema': rows[-1]['schema_version'] = 2
            elif mutation == 'missing':
                for row in rows: row['triggers'][0]['evidence']['line'] = 999999
            elif mutation == 'link': rows[-1]['related_incident_ids'] = ['missing']
        rewrite(partial, edit)
    with pytest.raises(ValidationError):
        load_session(partial)

@pytest.mark.parametrize('offset,expected_count', [(30000, 1), (120100, 2)])
def test_persisted_overlap_merge_or_link(go_command, tmp_path, offset, expected_count):
    import shutil
    executable, original_pack = go_command
    pack = tmp_path / 'pack'
    shutil.copytree(original_pack, pack)
    path = pack / 'scenarios/manual_mark.json'
    scenario = json.loads(path.read_text())
    second = dict(scenario['events'][0], offset_ms=offset)
    second['attributes'] = dict(second['attributes'], label='second marker')
    scenario['events'].append(second)
    path.write_text(json.dumps(scenario))
    directory = tmp_path / 'overlap'
    generate((executable, pack), directory, 'manual_mark')
    summary = load_session(directory)
    assert summary['incident_count'] == expected_count
    assert summary['trigger_count'] == 2
    if expected_count == 1:
        assert len(summary['incidents'][0]['triggers']) == 2
        assert summary['incidents'][0]['status'] == 'captured'
    else:
        a, b = summary['incidents']
        assert a['status'] == 'captured' and b['status'] == 'partial'
        assert a['related_incident_ids'] == [b['incident_id']]
        assert b['related_incident_ids'] == [a['incident_id']]
        assert a['window_end_unix_ms'] >= b['window_start_unix_ms']
    for item in summary['incidents']:
        assert reconstruct(directory, item['incident_id'])['incident'] == item
