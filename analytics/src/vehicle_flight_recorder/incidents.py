"""Append-only incident revisions and canonical evidence reconstruction."""

import heapq
import math
from pathlib import Path

from .session import ValidationError, _fail, _integer, _records, _schema, _string


def timeline(directory):
    """Merge canonical streams; new exports carry a total record sequence."""
    def stream(name):
        previous = (0, 0)
        for line, (_, record) in enumerate(_records(Path(directory) / name), 1):
            stamp = _integer(record.get('timestamp_unix_ms'), 'record timestamp', 1)
            sequence = _integer(record.get('record_sequence', 0), 'record_sequence')
            key = (stamp, sequence)
            if key < previous:
                _fail('canonical stream order changed')
            previous = key
            ref = dict(file=name, line=line, record_sequence=sequence, timestamp_unix_ms=stamp)
            field = 'signal_id' if name == 'observations.jsonl' else 'event_type'
            ref[field] = record[field]
            yield stamp, sequence, name, line, ref, record
    merged = heapq.merge(stream('observations.jsonl'), stream('events.jsonl'))
    mode = None
    for count, (_, sequence, _, _, ref, record) in enumerate(merged, 1):
        if mode is None:
            mode = sequence > 0
        if (mode and sequence != count) or (not mode and sequence != 0):
            _fail('record_sequence must be contiguous across canonical streams or entirely absent')
        yield ref, record


def _incident(item, metadata):
    _schema(item, 'incident')
    for key in ('incident_id', 'session_id', 'trigger_type'):
        _string(item.get(key), 'incident ' + key)
    if item['session_id'] != metadata['session_id']:
        _fail('incident session identity mismatch')
    _integer(item.get('revision'), 'incident revision', 1)
    for key in ('trigger_time_unix_ms', 'available_start_unix_ms', 'available_end_unix_ms', 'window_end_unix_ms'):
        _integer(item.get(key), 'incident ' + key, 1)
    start = item.get('window_start_unix_ms')
    if type(start) is not int or not -(2**63) <= start < 2**63:
        _fail('invalid incident window start')
    config = metadata.get('incident_capture')
    if not config:
        _fail('incident requires capture configuration')
    for key, config_key in [('pre_window_seconds', 'pre_window_ms'), ('post_window_seconds', 'post_window_ms')]:
        value = item.get(key)
        if type(value) not in (int, float) or not math.isfinite(value) or value * 1000 != config[config_key]:
            _fail('incident capture window differs from session configuration')
    if not isinstance(item.get('triggers'), list) or not item['triggers']:
        _fail('incident requires triggers')
    ranks = {'info': 1, 'warning': 2, 'error': 3}
    seen = set()
    last_sequence = 0
    for trigger in item['triggers']:
        if not isinstance(trigger, dict):
            _fail('invalid trigger object')
        for key in ('trigger_id', 'trigger_type', 'category'):
            _string(trigger.get(key), 'trigger ' + key)
        if trigger['trigger_id'] in seen or trigger.get('severity') not in ranks or type(trigger.get('manual')) is not bool:
            _fail('invalid or duplicate trigger')
        seen.add(trigger['trigger_id'])
        ref = trigger.get('evidence')
        if not isinstance(ref, dict) or ref.get('file') not in ('observations.jsonl', 'events.jsonl'):
            _fail('invalid trigger evidence file')
        _integer(ref.get('line'), 'evidence line', 1)
        sequence = _integer(ref.get('record_sequence'), 'evidence sequence', 1)
        stamp = _integer(trigger.get('timestamp_unix_ms'), 'trigger timestamp', metadata['start_time_unix_ms'])
        if sequence < last_sequence or stamp != ref.get('timestamp_unix_ms'):
            _fail('trigger evidence order/time mismatch')
        last_sequence = sequence
        if metadata['end_time_unix_ms'] is not None and stamp > metadata['end_time_unix_ms']:
            _fail('trigger exceeds session end')
        if 'details' in trigger and not isinstance(trigger['details'], dict):
            _fail('trigger details must be an object')
    first = item['triggers'][0]
    if item['trigger_time_unix_ms'] != first['timestamp_unix_ms'] or item['trigger_type'] != first['trigger_type']:
        _fail('primary trigger changed')
    if start != first['timestamp_unix_ms'] - config['pre_window_ms'] or item['window_end_unix_ms'] != max(t['timestamp_unix_ms'] for t in item['triggers']) + config['post_window_ms']:
        _fail('incident requested window does not match triggers')
    available_start, available_end = item['available_start_unix_ms'], item['available_end_unix_ms']
    if max(t['timestamp_unix_ms'] for t in item['triggers']) > available_end:
        _fail('trigger exceeds available evidence bounds')
    if not max(start, metadata['start_time_unix_ms']) <= available_start <= first['timestamp_unix_ms'] <= available_end <= item['window_end_unix_ms']:
        _fail('invalid available incident coverage')
    if metadata['end_time_unix_ms'] is not None and available_end > metadata['end_time_unix_ms']:
        _fail('coverage exceeds session end')
    if item.get('severity') != max((t['severity'] for t in item['triggers']), key=ranks.get) or item.get('manual') is not any(t['manual'] for t in item['triggers']):
        _fail('incident severity/manual aggregation mismatch')
    related = item.get('related_incident_ids')
    if not isinstance(related, list) or any(not isinstance(x, str) or not x or x == item['incident_id'] for x in related) or len(set(related)) != len(related):
        _fail('invalid incident links')
    gaps = item.get('coverage_gaps')
    if not isinstance(gaps, list):
        _fail('coverage_gaps must be an array')
    previous_end = available_start
    for gap in gaps:
        if not isinstance(gap, dict):
            _fail('invalid coverage gap')
        begin = _integer(gap.get('start_time_unix_ms'), 'gap start', previous_end)
        end = _integer(gap.get('end_time_unix_ms'), 'gap end', begin + 1)
        if end > available_end or gap.get('reason') not in ('transport_disconnected', 'record_silence'):
            _fail('invalid coverage gap bounds/reason')
        previous_end = end
    reasons = []
    if available_start > start:
        reasons.append('pre_window_unavailable')
    if available_end < item['window_end_unix_ms']:
        reasons.append('post_window_unavailable')
    if gaps:
        reasons.append('coverage_gap')
    if item.get('incomplete_reasons') != reasons or item.get('status') not in ('capturing', 'captured', 'partial'):
        _fail('invalid capture status/reasons')
    if item['status'] != 'capturing' and item['status'] != ('partial' if reasons else 'captured'):
        _fail('capture status contradicts available evidence')


def inspect_incidents(directory, metadata):
    latest = {}
    for location, item in _records(Path(directory) / 'incidents.jsonl'):
        try:
            _incident(item, metadata)
            old = latest.get(item['incident_id'])
            if item['revision'] != (old['revision'] + 1 if old else 1):
                _fail('incident revision must increment by one')
            if old:
                for key in ('session_id', 'trigger_time_unix_ms', 'trigger_type', 'pre_window_seconds', 'post_window_seconds', 'window_start_unix_ms', 'available_start_unix_ms'):
                    if old[key] != item[key]:
                        _fail('incident revision changed immutable identity/window')
                if item['triggers'][:len(old['triggers'])] != old['triggers'] or item['available_end_unix_ms'] < old['available_end_unix_ms'] or not set(old['related_incident_ids']) <= set(item['related_incident_ids']):
                    _fail('incident revision rewrote existing evidence')
                if old['status'] != 'capturing':
                    for key in ('status', 'triggers', 'available_end_unix_ms', 'window_end_unix_ms', 'coverage_gaps', 'incomplete_reasons'):
                        if old[key] != item[key]:
                            _fail('finalized capture changed')
            latest[item['incident_id']] = item
        except (KeyError, TypeError, ValueError) as error:
            raise ValidationError(f'{location}: {error}') from error
    references = {}
    trigger_ids = set()
    for item in latest.values():
        if metadata['state'] == 'complete' and item['status'] == 'capturing':
            _fail('complete session has unfinished incident capture')
        for related in item['related_incident_ids']:
            if related not in latest or item['incident_id'] not in latest[related]['related_incident_ids']:
                _fail('incident links must resolve symmetrically')
        for trigger in item['triggers']:
            if trigger['trigger_id'] in trigger_ids:
                _fail('trigger is duplicated across incidents')
            trigger_ids.add(trigger['trigger_id'])
            ref = trigger['evidence']
            references.setdefault((ref['file'], ref['line']), []).append(ref)
    for ref, record in timeline(directory):
        for expected in references.pop((ref['file'], ref['line']), []):
            if ref != expected:
                _fail('incident evidence reference does not match canonical source')
    if references:
        _fail('incident references missing canonical evidence')
    if metadata['state'] == 'complete' and metadata['incident_count'] != len(latest):
        _fail('finalized incident count does not match latest revisions')
    return list(latest.values())


def reconstruct(directory, incident_id, *, allow_incomplete=False):
    from .session import load_session
    summary = load_session(directory, allow_incomplete=allow_incomplete)
    item = next((i for i in summary['incidents'] if i['incident_id'] == incident_id), None)
    if item is None:
        _fail(f'unknown incident ID {incident_id!r}')
    rows = [dict(reference=ref, record=record) for ref, record in timeline(directory) if item['available_start_unix_ms'] <= ref['timestamp_unix_ms'] <= item['available_end_unix_ms']]
    return dict(schema_version=1, session_id=summary['session_id'], session_status=summary['status'], incident=item, timeline_order='timestamp_then_record_sequence', timeline=rows, interpretation='Synthetic temporal evidence; no causal diagnosis inferred')
