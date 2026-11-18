"""Independent replay of actual Go automatic/probe exports, all synthetic."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import pytest
from vehicle_flight_recorder.session import load_session, ValidationError, IncompleteSessionError
from vehicle_flight_recorder.incidents import reconstruct

@pytest.fixture(scope='module')
def automatic_exports(tmp_path_factory):
    root = Path(__file__).resolve().parents[2]
    output = tmp_path_factory.mktemp('automatic')
    binary = output/'automatic-fixture'
    env = os.environ | {'GOCACHE': str(root/'.cache/go-build'), 'GOMODCACHE': str(root/'.cache/go-mod')}
    subprocess.run(['go','build','-o',str(binary),'./cmd/automatic-fixture'], cwd=root/'core',env=env,check=True)
    result = {}
    for scenario in ('running','engine_off','manual_stop','interrupted','reconnect'):
        folder=output/scenario
        subprocess.run([str(binary),'--output',str(folder),'--scenario',scenario],check=True,capture_output=True,text=True)
        result[scenario]=folder
    return result

@pytest.mark.parametrize('scenario',['running','engine_off','manual_stop','interrupted','reconnect'])
def test_automatic_actual_exports(automatic_exports,scenario):
    root=automatic_exports[scenario]
    paths=sorted((root/'Sessions').glob('live-*'))
    if scenario=='engine_off': assert not paths
    else: assert len(paths)==1
    for path in paths:
        if scenario=='interrupted':
            with pytest.raises(IncompleteSessionError): load_session(path)
        result=load_session(path,allow_incomplete=scenario=='interrupted')
        assert result['live']['origin']=='scripted-fixture'
        assert result['observation_count']>0
        if scenario not in ('manual_stop','interrupted'):
            assert result['incident_count']==1
            assert result['incidents'][0]['status']=='captured'
            assert reconstruct(path,result['incidents'][0]['incident_id'])
        if scenario=='reconnect': assert result['live']['capability_report']['generation']==2
    probes=list((root/'Automation'/'Probes').glob('live-*'))
    assert probes
    for path in probes:
        result=load_session(path,allow_incomplete=scenario=='interrupted')
        assert set(result['signal_coverage']) <= {'powertrain.engine.rpm'}
    journal=[json.loads(x) for x in (root/'Automation'/'journal.jsonl').read_text().splitlines()]
    claims=[x for x in journal if x['event']=='confirmation_claimed']
    assert len(claims)==(0 if scenario=='engine_off' else 1)

@pytest.mark.parametrize('change',['policy','origin','budget','manual_report'])
def test_automatic_config_relabel_rejected(automatic_exports,tmp_path,change):
    source=next((automatic_exports['running']/'Sessions').glob('live-*'))
    path=tmp_path/'copy';shutil.copytree(source,path)
    meta=json.loads((path/'session.json').read_text());report=json.loads((path/'capability.json').read_text())
    if change=='policy': meta['live']['policy_version']='invented'
    elif change=='origin': meta['live']['start_origin']='activity_probe'
    elif change=='budget': report['configuration']['maximum_duration_ns']=86400*10**9
    else: report['configuration'].pop('policy_version');report['configuration'].pop('start_origin')
    (path/'session.json').write_text(json.dumps(meta));(path/'capability.json').write_text(json.dumps(report))
    with pytest.raises(ValidationError): load_session(path)

@pytest.mark.parametrize('scenario',['running','engine_off','manual_stop','interrupted','reconnect'])
def test_whole_automatic_bundle(automatic_exports,scenario):
    from vehicle_flight_recorder.automatic import validate_automatic
    result=validate_automatic(automatic_exports[scenario],allow_incomplete=scenario=='interrupted')
    assert result['drive_count']==(0 if scenario=='engine_off' else 1)
    assert result['probe_count']>=1 and result['owner_count']==1
    assert result['physical_acceptance'] is False
    for drive in result['drives']:
        runtime=drive['live']['runtime_quality']
        assert len(runtime['successful_automatic_start_facts'])==1
        if scenario=='reconnect': assert runtime['explicitly_logged_gap_seconds']>=9

@pytest.mark.parametrize('change',['duplicate_claim','foreign_claim','clock','origin'])
def test_journal_corruption_rejected(automatic_exports,tmp_path,change):
    from vehicle_flight_recorder.automatic import validate_automatic
    path=tmp_path/'bundle';shutil.copytree(automatic_exports['running'],path)
    file=path/'Automation'/'journal.jsonl';rows=[json.loads(x) for x in file.read_text().splitlines()]
    index=next(i for i,x in enumerate(rows) if x['event']=='confirmation_claimed')
    if change=='duplicate_claim': rows.insert(index+1,rows[index])
    elif change=='foreign_claim': rows[index]['details']['session_id']='live-'+'f'*32
    elif change=='clock': rows[-1]['native_ns']=1
    else: rows[index]['origin']='physical'
    file.write_text(''.join(json.dumps(x)+'\n' for x in rows))
    with pytest.raises(ValidationError):validate_automatic(path)

@pytest.mark.parametrize('change',['duplicate','foreign','manual_origin'])
def test_success_start_fact_is_checked_against_drive(automatic_exports,tmp_path,change):
    source=next((automatic_exports['running']/'Sessions').glob('live-*'))
    path=tmp_path/'copy';shutil.copytree(source,path)
    file=path/'events.jsonl';rows=[json.loads(x) for x in file.read_text().splitlines()]
    row=next(x for x in rows if x['event_type']=='automatic_recording_started')
    if change=='foreign':row['attributes']['session_id']='live-'+'f'*32
    elif change=='manual_origin':row['attributes']['start_origin']='manual'
    else:
        # Preserve counts/ordering to reach the independent duplicate-fact check.
        other=next(x for x in rows if x['event_type']=='automatic_activity_transition' and x['timestamp_unix_ms']>row['timestamp_unix_ms'])
        other['event_type']='automatic_recording_started';other['attributes']=row['attributes']
    file.write_text(''.join(json.dumps(x)+'\n' for x in rows))
    with pytest.raises(ValidationError):load_session(path)

def test_measured_freshness_populations_keep_initial_absence_and_denominators(automatic_exports):
    path=next((automatic_exports['reconnect']/'Sessions').glob('live-*'))
    result=load_session(path)
    quality=result['live']['runtime_quality']
    assert quality['explicitly_logged_gap_seconds']>=9
    for population in quality['receipt_freshness_populations']:
        assert population['count']>0
        assert population['stale_or_not_yet_observed_seconds']>0
        assert population['receipt_age_fresh_seconds']+population['stale_or_not_yet_observed_seconds']==pytest.approx(quality['total_duration_seconds'])
        assert 0<=population['receipt_fresh_outside_explicit_gaps_seconds']<=population['receipt_age_fresh_seconds']
        assert population['observations_outside_explicit_gaps']<=population['count']
        assert population['achieved_whole_interval_hz']==pytest.approx(population['count']/quality['total_duration_seconds'])

@pytest.mark.parametrize('target',['state','journal','transition'])
def test_wrong_runtime_field_types_fail_cleanly(automatic_exports,tmp_path,target):
    from vehicle_flight_recorder.automatic import validate_automatic
    path=tmp_path/'bundle';shutil.copytree(automatic_exports['running'],path)
    if target=='state': (path/'Automation'/'state.json').write_text('[]')
    elif target=='journal':
        file=path/'Automation'/'journal.jsonl';rows=[json.loads(x) for x in file.read_text().splitlines()]
        rows[0]['origin']=[];file.write_text(''.join(json.dumps(x)+'\n' for x in rows))
    else:
        drive=next((path/'Sessions').glob('live-*'));file=drive/'events.jsonl';rows=[json.loads(x) for x in file.read_text().splitlines()]
        next(x for x in rows if x['event_type']=='automatic_activity_transition')['attributes']['state']=[]
        file.write_text(''.join(json.dumps(x)+'\n' for x in rows))
    with pytest.raises(ValidationError):validate_automatic(path)
