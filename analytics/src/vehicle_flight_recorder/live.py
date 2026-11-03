"""Schema-2 integrity cross-checks against the independently replayed live report."""
import base64
from pathlib import Path
from .session import ValidationError, _decode, _integer, _records
from .acquisition import fields, require, validate_acquisition


class LiveEvidence:
    def __init__(self, directory, metadata):
        self.directory, self.metadata = Path(directory), metadata
        required = {'session.json', 'observations.jsonl', 'events.jsonl', 'incidents.jsonl', 'raw/transport.jsonl', 'capability.json'}
        actual = {p.relative_to(self.directory).as_posix() for p in self.directory.rglob('*') if p.is_file()}
        extras = actual-required
        self.incomplete = metadata['state'] != 'complete'
        self.temporary = sorted(extras) if self.incomplete and all(x.startswith('.capability-') and '/' not in x for x in extras) else []
        require(actual-set(self.temporary) == required, 'unknown/missing live evidence files')
        for name in required:
            p = self.directory/name
            require(not p.is_symlink() and p.resolve() == self.directory.resolve()/name, 'nonlocal evidence file')
            require(p.stat().st_size <= (16*1024*1024 if name == 'capability.json' else 64*1024*1024), 'file budget')
        self.summary = validate_acquisition(self.directory/'capability.json', allow_incomplete=self.incomplete)
        with (self.directory/'capability.json').open('rb') as f:
            self.report = _decode(f.read(16*1024*1024+1), 'capability.json')
        require(self.report['report_version'] == 2, 'live session requires report 2')
        config, live = self.report['configuration'], metadata['live']
        require(config['session_id'] == metadata['session_id'] and self.report['anchor_unix_ms'] == metadata['start_time_unix_ms'], 'foreign report/session anchor')
        for key in ('origin', 'operating_state', 'workload', 'native_anchor_ns'):
            require(config[key] == live[key], 'session/report configuration mismatch')
        self.transactions = {x['id']: x for x in self.report['transactions']}
        self.observations = {(o['transaction_id'], o['generation'], o['ecu'], o['signal_id']): o for o in self.report['observations']}
        require(len(self.observations) == len(self.report['observations']), 'duplicate report observation')
        self.seen = set()
        self.tail = 0
        previous_ns = 0
        rows = 0
        for index, (_, row) in enumerate(_records(self.directory/'raw/transport.jsonl')):
            rows += 1
            if index < len(self.report['raw']):
                require(row == self.report['raw'][index], 'raw ledger differs from authoritative report checkpoint')
            else:
                require(self.incomplete, 'complete session has uncheckpointed raw evidence')
                fields(row, 'sequence generation transaction_id kind monotonic_ns bytes_base64 session_id')
                require(row['session_id'] == metadata['session_id'] and row['sequence'] == index+1, 'foreign/truncated raw tail')
                _integer(row['generation'], 'raw tail generation', 1)
                ns = _integer(row['monotonic_ns'], 'raw tail clock')
                require(ns >= previous_ns and row['kind'] in {'requested','accepted','received','late_received','unsolicited'}, 'invalid raw tail')
                require(isinstance(row['transaction_id'], str) and isinstance(row['bytes_base64'], str), 'invalid raw tail fields')
                try: data = base64.b64decode(row['bytes_base64'], validate=True)
                except (ValueError, TypeError) as e: raise ValidationError('invalid raw tail Base64') from e
                require(0 < len(data) <= 65536, 'raw tail byte budget')
                self.tail += 1
            previous_ns = row['monotonic_ns']
        require(rows >= len(self.report['raw']) and rows <= 16384, 'missing/oversized raw ledger')
        if not self.incomplete:
            require(self.report['state'] == 'disconnected' and self.report['duration_ns']//1000000 == metadata['end_time_unix_ms']-metadata['start_time_unix_ms'], 'live finalization/report mismatch')

    def observation(self, row):
        allowed = {'schema_version','record_sequence','timestamp_unix_ms','session_id','signal_id','source','value','unit','quality','module','bus','decoder_version','vehicle_pack_version','provenance'}
        require(set(row) == allowed, 'unknown/missing canonical live observation fields')
        p = row['provenance']
        fields(p, 'transaction_id generation raw_refs request_start_ns stream_accepted_ns receive_ns measurement_unix_ms')
        identity = (p['transaction_id'], p['generation'], row['module'], row['signal_id'])
        require(identity in self.observations and identity not in self.seen, 'foreign/duplicate canonical observation')
        expected = self.observations[identity]
        tx = self.transactions[p['transaction_id']]
        for key in ('value','unit','decoder_version'):
            require(row[key] == expected[key], 'canonical decoder value differs from raw provenance')
        require(row['timestamp_unix_ms'] == expected['mapped_unix_ms'] and row['quality'] == 'valid' and row['bus'] == 'HS-CAN' and row['vehicle_pack_version'] == 'none', 'invented canonical quality/time/source')
        require(type(p['generation']) is int and p['raw_refs'] == tx['raw_refs'] and p['request_start_ns'] == tx['start_ns'] and p['stream_accepted_ns'] == tx['sent_ns'] and p['receive_ns'] == expected['receive_ns'] and p['measurement_unix_ms'] is None, 'broken transaction reference or invented measurement time')
        for key in ('generation','request_start_ns','stream_accepted_ns','receive_ns'): _integer(p[key], key)
        require(isinstance(p['raw_refs'], list) and all(type(x) is int for x in p['raw_refs']), 'raw reference types')
        self.seen.add(identity)

    def finish(self):
        if not self.incomplete: require(self.seen == set(self.observations), 'canonical export omitted report observations')
        return dict(capability_report=self.summary, origin=self.metadata['live']['origin'], uncheckpointed_raw_rows=self.tail,
                    uncommitted_report_observations=len(self.observations)-len(self.seen), interrupted_checkpoint_files=self.temporary)
