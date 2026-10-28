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
if len(archives) < 2:
    raise SystemExit('Expected native healthy/manual and voltage/TCM ZIP exports')
expected = {'session.json', 'observations.jsonl', 'events.jsonl', 'incidents.jsonl', 'raw/transport.jsonl'}
scenarios = set()
results = []
for archive in archives:
    destination = root / 'unpacked' / archive.stem
    with zipfile.ZipFile(archive) as source:
        if set(source.namelist()) != expected or len(source.namelist()) != len(expected) or source.testzip() is not None:
            raise SystemExit(f'Invalid native ZIP layout/integrity: {archive}')
        source.extractall(destination)
    metadata = json.loads((destination / 'session.json').read_text())
    scenarios.add(metadata['mock']['scenario'])
    summary = load_session(destination)
    if summary['state'] != 'complete' or summary['incident_count'] != 1:
        raise SystemExit('Native test export must have one finalized incident')
    incident = summary['incidents'][0]
    if incident['status'] != 'captured':
        raise SystemExit('Native scenario did not capture full evidence')
    reconstruction = reconstruct(destination, incident['incident_id'])
    (destination / 'reconstructed-incident.json').write_text(json.dumps(reconstruction, indent=2) + '\n')
    results.append(summary)
if not {'healthy_drive', 'voltage_tcm_dropout'} <= scenarios:
    raise SystemExit('Native test scenario coverage is incomplete')
(root / 'python-validation.json').write_text(json.dumps(results, indent=2) + '\n')
print(f'Validated {len(archives)} real native ZIP exports and reconstructed their incidents')
