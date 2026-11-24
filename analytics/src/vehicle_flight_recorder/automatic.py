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
            require(isinstance(attrs.get('state'),str) and attrs['state'] in {'waiting_for_activity','candidate','starting','recording','end_grace','degraded_gap','finalized','manual_stop_override'},'invalid activity transition state')
            states[state]+=max(0,at-state_start);state=attrs['state'];state_start=at
        if kind=='automatic_recording_started':
            fields(attrs,'session_id start_origin time_unix_ms vehicle_name coverage')
            require(metadata['live']['start_origin']=='automatic' and attrs['session_id']==metadata['session_id'] and attrs['start_origin']=='automatic' and attrs['coverage']=='limited_standard_obd','foreign/invented automatic start fact')
            require(isinstance(attrs['vehicle_name'],str) and 0<len(attrs['vehicle_name'].encode())<=128,'invalid display identity')
            require(_integer(attrs['time_unix_ms'],'automatic fact time',1)==at,'success fact clock differs from canonical mapping')
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
    # Receipt-age freshness is a reproducible UI convention, not an ECU sensor
    # refresh or measurement-time claim. Initial absent time is included.
    populations={}
    schedules={x['key']:x for x in report['schedule']}
    signal_keys={'powertrain.engine.rpm':'rpm','powertrain.coolant.temperature':'coolant','electrical.adapter_voltage':'adapter_voltage'}
    for observation in report['observations']:
        populations.setdefault((observation['signal_id'],observation['ecu']),[]).append(observation['mapped_unix_ms'])
    freshness=[]
    for (signal,ecu),times in sorted(populations.items()):
        windows=[]
        for at in times:
            interval=(max(start,at),min(end,at+15000))
            if interval[1]<=interval[0]:continue
            if windows and interval[0]<=windows[-1][1]:windows[-1]=(windows[-1][0],max(interval[1],windows[-1][1]))
            else:windows.append(interval)
        fresh=sum(b-a for a,b in windows)
        overlap=sum(max(0,min(b,d)-max(a,c)) for a,b in windows for c,d in merged)
        outside_count=sum(not any(a<=at<b for a,b in merged) for at in times)
        whole_seconds=(end-start)/1000
        outside_seconds=(end-start-excluded)/1000
        schedule=schedules[signal_keys[signal]]
        freshness.append(dict(signal_id=signal,ecu=ecu,count=len(times),requested_interval_seconds=schedule['interval_ns']/10**9,
            achieved_whole_interval_hz=len(times)/whole_seconds if whole_seconds else None,
            observations_outside_explicit_gaps=outside_count,achieved_outside_explicit_gaps_hz=outside_count/outside_seconds if outside_seconds>0 else None,
            receipt_freshness_budget_seconds=15,receipt_age_fresh_seconds=fresh/1000,
            stale_or_not_yet_observed_seconds=max(0,end-start-fresh)/1000,
            receipt_fresh_outside_explicit_gaps_seconds=max(0,fresh-overlap)/1000))
    return dict(total_duration_seconds=(end-start)/1000, explicitly_logged_gap_seconds=excluded/1000,
        interval_outside_explicit_gaps_seconds=max(0,end-start-excluded)/1000,
        interval_caveat='Not proof of continuous execution, fresh coverage, or ECU sensor refresh',
        receipt_freshness_populations=freshness,
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
    require(isinstance(state,dict) and required<=set(state)<=required|{'active_drive','last_failure','quiet_after_stop','resume_session_id','resume_reason','pending_completion'} and state['version']==POLICY,'unsupported automatic state')
    require(all(type(state[k]) is bool for k in ('enabled','manual_stop_override','confirmation_claimed')) and state['protocol_profile'] in {'can11-single-frame-headers-pci','can29-normal-fixed-single-frame-headers-pci'},'state policy/profile types')
    if 'last_failure' in state:require(isinstance(state['last_failure'],str) and len(state['last_failure'].encode())<=512,'invalid persisted failure')
    if 'active_drive' in state:require(isinstance(state['active_drive'],str) and ID.fullmatch(state['active_drive']),'state drive identity')
    reasons={'rpm_unavailable','scanner_disconnected','three_zero_rpm_replies','owner_execution_gap_exceeded_grace','controlled_recording_duration_limit'}
    if 'quiet_after_stop' in state:
        require(type(state['quiet_after_stop']) is bool,'invalid quiet state')
    if 'resume_session_id' in state or 'resume_reason' in state:
        require(state.get('quiet_after_stop') is True and isinstance(state.get('resume_session_id'),str) and ID.fullmatch(state['resume_session_id']) and isinstance(state.get('resume_reason'),str) and state['resume_reason'] in reasons,'invalid drive resume state')
    if 'pending_completion' in state:
        completion=state['pending_completion']
        require(isinstance(completion,dict) and set(completion)=={'session_id','reason'} and isinstance(completion['session_id'],str) and ID.fullmatch(completion['session_id']) and isinstance(completion['reason'],str) and completion['reason'] in reasons,'invalid completion receipt')
    journal=root/'Automation'/'journal.jsonl' 
    require(journal.is_file() and not journal.is_symlink() and journal.stat().st_size<=16*1024*1024+8193,'missing/nonlocal/oversized automation journal')
    counts=Counter(); owners={}; claims={}; resources={}; batteries={}
    current=None
    known={'owner_opened','activity_transition','confirmation_claimed','automatic_preference_changed','interrupted_session_preserved','bounded_session_ended','manual_stop_override','resource_sample','foreground','background','inactive','confirmation_result','battery_sample','notification_tap','monitoring_failed','failed_capture_preserved','accessory_directory','export_requested','export_result'}
    for _,row in _records(journal):
        fields(row,'policy_version owner_id origin observed_wall_unix_ms event native_ns phase details')
        require(all(isinstance(row[k],str) for k in ('owner_id','origin','event','phase','policy_version')),'journal string field types')
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
