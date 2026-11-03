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
if len(archives) < 3:
    raise SystemExit('Expected native healthy/manual, voltage/TCM and live-fixture ZIP exports')
expected = {'session.json', 'observations.jsonl', 'events.jsonl', 'incidents.jsonl', 'raw/transport.jsonl'}
scenarios = set()
results = []
live_count = 0
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
    if summary['state'] != 'complete' or summary['incident_count'] != 1:
        raise SystemExit('Native test export must have one finalized incident')
    incident = summary['incidents'][0]
    if incident['status'] != 'captured':
        raise SystemExit('Native scenario did not capture full evidence')
    reconstruction = reconstruct(destination, incident['incident_id'])
    reconstructed = root / 'reconstructions'
    reconstructed.mkdir(exist_ok=True)
    (reconstructed / f'{archive.stem}.json').write_text(json.dumps(reconstruction, indent=2) + '\n')
    results.append(summary)
if live_count < 1 or not {'healthy_drive', 'voltage_tcm_dropout'} <= scenarios:
    raise SystemExit('Native test scenario coverage is incomplete')
(root / 'python-validation.json').write_text(json.dumps(results, indent=2) + '\n')
print(f'Validated {len(archives)} real native ZIP exports and reconstructed their incidents')
