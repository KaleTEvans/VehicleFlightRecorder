"""Exercise Python against actual Go reports; mutations must fail validation."""
import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import pytest
from vehicle_flight_recorder.acquisition import validate_acquisition
from vehicle_flight_recorder.session import ValidationError


@pytest.fixture(scope='module')
def acquisition_reports(tmp_path_factory):
    root = Path(__file__).resolve().parents[2]
    output = tmp_path_factory.mktemp('acquisition')
    go = shutil.which('go')
    assert go, 'Go is required for acquisition integration tests'
    env = os.environ | {'GOCACHE': str(root / '.cache/go-build'), 'GOMODCACHE': str(root / '.cache/go-mod')}
    binary = output / 'offline-acquisition'
    subprocess.run([go, 'build', '-o', str(binary), './cmd/offline-acquisition'], cwd=root / 'core', env=env, check=True)
    reports = {}
    for scenario in ['healthy', 'edge_cases']:
        first, second = output / f'{scenario}.json', output / f'{scenario}-repeat.json'
        for path in [first, second]:
            subprocess.run([str(binary), '--scenario', scenario, '--output', str(path)], check=True, capture_output=True)
        assert first.read_bytes() == second.read_bytes()
        failed = subprocess.run([str(binary), '--output', str(first)], capture_output=True)
        assert failed.returncode != 0
        reports[scenario] = first
    return reports


def test_actual_go_reports_and_cli(acquisition_reports):
    for scenario, path in acquisition_reports.items():
        result = validate_acquisition(path)
        assert result['physical_validation'] is False
        assert result['source'] == 'synthetic'
        assert result['duration_seconds'] == 12
        assert result['observation_count'] == (51 if scenario == 'healthy' else 46)
        assert result['generation'] == (1 if scenario == 'healthy' else 2)
        if scenario == 'edge_cases': assert result['outcomes']['timeout'] == 1
        cli = subprocess.run([sys.executable, '-m', 'vehicle_flight_recorder', 'validate-acquisition', str(path)], capture_output=True, check=True)
        assert json.loads(cli.stdout) == result


@pytest.mark.parametrize('mutation', ['version', 'source', 'duration', 'generation', 'raw_ref', 'raw_sequence', 'command', 'bytes', 'nan', 'zero_as_missing', 'decoder', 'value', 'time', 'measurement_time', 'rate', 'count', 'gap', 'latency', 'capability', 'unsupported', 'duplicate_observation', 'missing_observation', 'foreign_ecu', 'unknown_field', 'wrong_type'])
def test_corruption_rejected(acquisition_reports, tmp_path, mutation):
    report = json.loads(acquisition_reports['healthy'].read_text())
    if mutation == 'version': report['report_version'] = 2
    if mutation == 'source': report['source'] = 'live'
    if mutation == 'duration': report['duration_ns'] = 0
    if mutation == 'generation': report['transactions'][0]['generation'] = 100
    if mutation == 'raw_ref': report['transactions'][0]['raw_refs'] = [999]
    if mutation == 'raw_sequence': report['raw'][0]['sequence'] = 2
    if mutation == 'command': report['raw'][0]['bytes_base64'] = 'QVRaDQ=='
    if mutation == 'bytes': report['raw'][0]['bytes_base64'] = '!!!'
    if mutation == 'nan': report['observations'][0]['value'] = float('nan')
    if mutation == 'zero_as_missing': report['observations'][0]['value'] = None
    if mutation == 'decoder': report['observations'][0]['decoder_version'] = '2'
    if mutation == 'value': report['observations'][0]['value'] = 999
    if mutation == 'time': report['observations'][0]['receive_ns'] += 1
    if mutation == 'measurement_time': report['observations'][0]['measurement_unix_ms'] = 12345
    if mutation == 'rate': report['statistics'][0]['rate_hz'] *= 2
    if mutation == 'count': report['statistics'][0]['count'] += 1
    if mutation == 'gap': report['statistics'][0]['maximum_gap_ns'] = 0
    if mutation == 'latency': report['statistics'][0]['latency_p95_ns'] += 1
    if mutation == 'capability': report['capabilities'][0]['evidence_transaction'] = 'foreign'
    if mutation == 'unsupported': report['capabilities'][0]['advertised'] = False
    if mutation == 'duplicate_observation': report['observations'].append(copy.deepcopy(report['observations'][0]))
    if mutation == 'missing_observation': report['observations'].pop()
    if mutation == 'foreign_ecu': report['observations'][0]['ecu'] = 'TCM'
    if mutation == 'unknown_field': report['ignored_bytes'] = True
    if mutation == 'wrong_type': report['schedule'][0]['key'] = []
    path = tmp_path / 'bad.json'
    path.write_text(json.dumps(report))
    with pytest.raises(ValidationError): validate_acquisition(path)


def test_truncated_duplicate_and_unknown_fields(acquisition_reports, tmp_path):
    data = acquisition_reports['healthy'].read_bytes()
    path = tmp_path / 'bad.json'
    path.write_bytes(data[:-5])
    with pytest.raises(ValidationError): validate_acquisition(path)
    path.write_bytes(data.replace(b'"report_version": 1', b'"report_version": 1, "report_version": 1', 1))
    with pytest.raises(ValidationError): validate_acquisition(path)


def test_growth_after_size_check_is_bounded(tmp_path):
    from types import SimpleNamespace
    from unittest.mock import patch
    path = tmp_path / 'grown.json'
    path.write_bytes(b' ' * (8 * 1024 * 1024 + 1))
    with patch.object(Path, 'stat', return_value=SimpleNamespace(st_size=0)):
        with pytest.raises(ValidationError, match='grew beyond'):
            validate_acquisition(path)
