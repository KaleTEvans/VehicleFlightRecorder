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
