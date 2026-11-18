"""Read preserved M7 evidence without repair or implying physical acceptance."""
import json
from collections import Counter
from pathlib import Path
import re
from .session import ValidationError, _decode, _integer, _records, load_session
from .acquisition import fields, require

POLICY='rpm-activity-1'
ID=re.compile(r'live-[0-9a-f]{32}\Z')
OWNER=re.compile(r'owner-[0-9a-f]{32}\Z')


def quality(metadata, report, events):
    """Measured intervals exclude only explicitly logged gaps; unknown stays unknown."""
    start=metadata['start_time_unix_ms']
    end=metadata['end_time_unix_ms'] or max([start+report['duration_ns']//10**6]+[e['timestamp_unix_ms'] for e in events])
    gaps=[]; disconnected=None; phases=Counter(); states=Counter(); phase='unknown'; state='initializing'; phase_start=state_start=start
    starts=[]; confirmations=[]
    for event in events:
        kind=event['event_type']; at=event['timestamp_unix_ms']; attrs=event.get('attributes',{})
        if kind=='transport_disconnected' and disconnected is None: disconnected=at
        if kind=='transport_connected' and disconnected is not None:
            gaps.append((disconnected,at,'transport_disconnected'));disconnected=None
        if kind=='acquisition_pause_observed':
            fields(attrs,'from_ns to_ns cause')
            a=_integer(attrs['from_ns'],'pause start');b=_integer(attrs['to_ns'],'pause end')
            require(a<=b,'inverted pause')
            gaps.append((start+a//10**6,start+b//10**6,'unknown_native_scheduling_gap'))
        if kind in ('native_foreground','native_background','native_inactive'):
            phases[phase]+=max(0,at-phase_start);phase=kind.removeprefix('native_');phase_start=at
        if kind=='automatic_activity_transition':
            require(metadata['live']['start_origin']=='automatic','automatic event in manual/probe session')
            states[state]+=max(0,at-state_start);state=attrs.get('state','unknown');state_start=at
        if kind=='automatic_recording_started':
            fields(attrs,'session_id start_origin time_unix_ms vehicle_name coverage')
            require(metadata['live']['start_origin']=='automatic' and attrs['session_id']==metadata['session_id'] and attrs['start_origin']=='automatic' and attrs['coverage']=='limited_standard_obd','foreign/invented automatic start fact')
            require(isinstance(attrs['vehicle_name'],str) and 0<len(attrs['vehicle_name'].encode())<=128,'invalid display identity')
            _integer(attrs['time_unix_ms'],'automatic fact time',1)
            eligible=[o for o in report['observations'] if o['mapped_unix_ms']<=at]
            init=[t for t in report['transactions'] if t['key'].startswith('init') and t['outcome']=='ok' and t['end_ns'] is not None and start+t['end_ns']//10**6<=at]
            require(eligible and init,'success confirmation lacks accepted telemetry/setup')
            starts.append(attrs)
        if kind=='automatic_confirmation_result': confirmations.append(attrs)
    require(len(starts)<=1,'duplicate automatic successful-start fact')
    if disconnected is not None:gaps.append((disconnected,end,'transport_disconnected'))
    phases[phase]+=max(0,end-phase_start);states[state]+=max(0,end-state_start)
    intervals=sorted((max(start,a),min(end,b)) for a,b,_ in gaps if min(end,b)>max(start,a))
    merged=[]
    for a,b in intervals:
        if merged and a<=merged[-1][1]: merged[-1]=(merged[-1][0],max(b,merged[-1][1]))
        else:merged.append((a,b))
    excluded=sum(b-a for a,b in merged)
    return dict(total_duration_seconds=(end-start)/1000, explicitly_logged_gap_seconds=excluded/1000,
        interval_outside_explicit_gaps_seconds=max(0,end-start-excluded)/1000,
        interval_caveat='Not proof of continuous execution, fresh coverage, or ECU sensor refresh',
        gaps=[dict(start_unix_ms=a,end_unix_ms=b,reason=reason) for a,b,reason in gaps],
        observed_scene_intervals_seconds={k:v/1000 for k,v in phases.items()},
        activity_intervals_seconds={k:v/1000 for k,v in states.items()},
        successful_automatic_start_facts=starts,confirmation_outcomes=confirmations)


def validate_automatic(directory, allow_incomplete=False):
    root=Path(directory)
    state_path=root/'Automation'/'state.json'
    require(state_path.is_file() and not state_path.is_symlink() and state_path.stat().st_size<=8192,'missing/nonlocal/oversized automatic state')
    state=_decode(state_path.read_bytes(),'automatic state')
    required={'version','enabled','manual_stop_override','confirmation_claimed','protocol_profile'}
    require(required<=set(state)<=required|{'active_drive'} and state['version']==POLICY,'unsupported automatic state')
    require(all(type(state[k]) is bool for k in ('enabled','manual_stop_override','confirmation_claimed')) and state['protocol_profile'] in {'can11-single-frame-headers-pci','can29-normal-fixed-single-frame-headers-pci'},'state policy/profile types')
    if 'active_drive' in state:require(isinstance(state['active_drive'],str) and ID.fullmatch(state['active_drive']),'state drive identity')
    journal=root/'Automation'/'journal.jsonl' 
    require(journal.is_file() and not journal.is_symlink() and journal.stat().st_size<=16*1024*1024+8193,'missing/nonlocal/oversized automation journal')
    counts=Counter(); owners={}; claims={}; resources={}; batteries={}
    current=None
    known={'owner_opened','activity_transition','confirmation_claimed','automatic_preference_changed','interrupted_session_preserved','bounded_session_ended','manual_stop_override','resource_sample','foreground','background','inactive','confirmation_result','battery_sample','notification_tap'}
    for _,row in _records(journal):
        fields(row,'policy_version owner_id origin observed_wall_unix_ms event native_ns phase details')
        owner=row['owner_id'];require(isinstance(owner,str) and OWNER.fullmatch(owner),'owner identity')
        require(row['policy_version']==POLICY and row['origin'] in {'physical','native-fixture','scripted-fixture'} and row['event'] in known and row['phase'] in {'unknown','foreground','background','inactive'},'journal envelope')
        ns=_integer(row['native_ns'],'journal native clock');_integer(row['observed_wall_unix_ms'],'observed wall time',1)
        require(isinstance(row['details'],dict),'journal detail object')
        if owner!=current:
            require(owner not in owners and row['event']=='owner_opened','owner scope must start once')
            owners[owner]=dict(last_ns=ns,origin=row['origin']);current=owner
        require(ns>=owners[owner]['last_ns'] and row['origin']==owners[owner]['origin'],'journal clock/origin changed inside owner scope')
        owners[owner]['last_ns']=ns;counts[row['event']]+=1
        detail=row['details']
        if row['event']=='confirmation_claimed':
            identity=detail.get('session_id');require(isinstance(identity,str) and ID.fullmatch(identity) and detail.get('start_origin')=='automatic' and identity not in claims,'duplicate/invalid confirmation claim')
            claims[identity]=detail
        if row['event']=='resource_sample':
            for field in ('retained_bytes','go_heap_alloc_bytes','go_heap_objects'):_integer(detail.get(field),field)
            resources.setdefault(owner,[]).append((ns,detail))
        if row['event']=='battery_sample':
            try:sample=json.loads(detail['detail'])
            except (KeyError,TypeError,ValueError) as e:raise ValidationError('invalid battery sample') from e
            level=sample.get('battery_level')
            require(level is None or (type(level) in (int,float) and 0<=level<=1),'battery level')
            require(type(sample.get('battery_state')) is int and sample['battery_state'] in range(4),'battery charging state')
            batteries.setdefault(owner,[]).append((ns,sample))
    require(owners,'empty automation journal')
    drives=[];probes=[]
    for directory,purpose,results in [(root/'Sessions','automatic',drives),(root/'Automation'/'Probes','activity_probe',probes)]:
        if not directory.exists():continue
        for path in sorted(directory.iterdir()):
            if not path.is_dir() or not ID.fullmatch(path.name):continue
            meta=_decode((path/'session.json').read_bytes(),'session.json')
            if purpose=='automatic' and meta.get('live',{}).get('policy_version')!=POLICY:continue
            require(not path.is_symlink() and meta.get('live',{}).get('start_origin')==purpose,'probe/drive namespace mismatch')
            result=load_session(path,allow_incomplete=allow_incomplete)
            results.append(result)
    by_id={d['session_id']:d for d in drives}
    for identity in claims:
        require(identity in by_id and by_id[identity]['events_by_type'].get('automatic_recording_started')==1,'claimed confirmation missing successful drive fact')
    measurements=[]
    for owner in owners:
        item=dict(owner_id=owner,origin=owners[owner]['origin'])
        samples=resources.get(owner,[])
        if len(samples)>=2:
            seconds=(samples[-1][0]-samples[0][0])/10**9
            growth=samples[-1][1]['retained_bytes']-samples[0][1]['retained_bytes']
            item.update(resource_observed_seconds=seconds,retained_growth_bytes=growth,retained_growth_bytes_per_hour=growth*3600/seconds if seconds else None,maximum_observed_go_heap_bytes=max(s['go_heap_alloc_bytes'] for _,s in samples))
        samples=batteries.get(owner,[])
        if samples:
            item.update(battery_samples=len(samples),first_battery_sample=samples[0][1],last_battery_sample=samples[-1][1])
        if len(samples)>=2:
            seconds=(samples[-1][0]-samples[0][0])/10**9
            eligible=seconds>0 and all(s['battery_state']==1 and s['battery_level'] is not None for _,s in samples)
            item.update(battery_observed_seconds=seconds,observed_battery_fraction_per_hour=(samples[0][1]['battery_level']-samples[-1][1]['battery_level'])*3600/seconds if eligible else None,
                battery_caveat='Only sampled execution intervals; record screen/lock, charging, ambient and drive context separately. Not battery acceptance.')
        measurements.append(item)
    return dict(status='valid',policy_version=POLICY,physical_acceptance=False,owner_count=len(owners),journal_event_counts=dict(counts),confirmation_claim_count=len(claims),drive_count=len(drives),probe_count=len(probes),drives=drives,probes=probes,resource_measurements=measurements)
