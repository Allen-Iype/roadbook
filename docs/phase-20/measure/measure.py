"""Phase 20 (issue #8) measurement: which away-spans the destination rule rejects,
and what each candidate rule would admit.

A measurement port of prototype/detect_fixture.py: parsing, outlier rejection,
home bases and away-spans are the reference's, line for line; what is added is
bookkeeping about each span (which base it left and came back to, how long it
was continuously beyond FAR_KM, its visits and modes).

Reads its inputs read-only and writes nothing. By default it prints counts
only. --list also prints one line per span the recommended rule would add
(a date, durations and distances from home; never a coordinate) for the
person whose data it is to recognise; do not paste that output anywhere public.

    python3 docs/phase-20/measure/measure.py data/fixture-2025-04.segments.timeline.json \
        data/Timeline.json data/ios-timeline.as-android.json testdata/demo/demo.json

Inputs are Android-shape exports (an object with semanticSegments). The iPhone
export is measured through the same converted copy phase 17's regression uses.
"""
import json, math, collections, bisect, sys
from datetime import datetime, timedelta
import statistics as st
P=dict(NEAR_M=25000, FAR_KM=100, MIN_OBS=5, MIN_HRS=1.0, MIN_DWELL_MIN=60, MAX_KMH=900)
def hav(a,b):
    R=6371008.8;p1,p2=math.radians(a[0]),math.radians(b[0]);dp=p2-p1;dl=math.radians(b[1]-a[1])
    h=math.sin(dp/2)**2+math.cos(p1)*math.cos(p2)*math.sin(dl/2)**2
    return 2*R*math.asin(math.sqrt(h))
def ll(s):
    if isinstance(s,dict): s=s.get('latLng')
    if not s: return None
    p=s.replace('°','').split(',')
    try: return [float(p[0]),float(p[1])]
    except: return None
def ts(s): return datetime.fromisoformat(s)

def load(SRC):
    raw=json.load(open(SRC))
    visits=[];acts=[];paths=[];mem=[]
    for s in raw['semanticSegments']:
        a,b=s.get('startTime'),s.get('endTime')
        if 'visit' in s:
            tc=s['visit'].get('topCandidate',{})
            visits.append(dict(s=ts(a),e=ts(b),ll=ll(tc.get('placeLocation')),type=tc.get('semanticType')))
        elif 'activity' in s:
            ac=s['activity']
            acts.append(dict(s=ts(a),e=ts(b),a=ll(ac.get('start')),b=ll(ac.get('end')),
                             d=ac.get('distanceMeters') or 0,t=(ac.get('topCandidate') or {}).get('type')))
        elif 'timelinePath' in s:
            for pt in s['timelinePath']:
                if pt.get('time'): paths.append(dict(t=ts(pt['time']),ll=ll(pt.get('point'))))
        elif 'timelineMemory' in s:
            m=s['timelineMemory']; m=m.get('trip',m)
            try: km=float(m.get('distanceFromOriginKms'))
            except: km=None
            mem.append(dict(s=ts(a),e=ts(b),km=km))
    return visits,acts,paths,mem

def spans_of(visits,acts,paths):
    obs=[]
    for v in visits:
        if v['ll']: obs.append((v['s'],v['e'],v['ll'],'v'))
    for a in acts:
        if a['a']: obs.append((a['s'],a['s'],a['a'],'a'))
        if a['b']: obs.append((a['e'],a['e'],a['b'],'a'))
    for p in paths:
        if p['ll']: obs.append((p['t'],p['t'],p['ll'],'p'))
    obs.sort(key=lambda x:x[0])
    def bad(i):
        o=obs[i];n=c=0
        for j in (i-1,i+1):
            if 0<=j<len(obs):
                dt=abs((obs[j][0]-o[0]).total_seconds())/3600
                if dt<=0: continue
                c+=1
                if hav(o[2],obs[j][2])/1000/dt>P['MAX_KMH']: n+=1
        return c>0 and n==c
    flags=[bad(i) for i in range(len(obs))]
    nb=sum(flags); obs=[o for i,o in enumerate(obs) if not flags[i]]
    hv=[(v['s'].date(),v['ll']) for v in visits if v['type']=='INFERRED_HOME' and v['ll']]
    g=collections.defaultdict(list)
    for d,x in hv: g[(round(x[0]*20)/20,round(x[1]*20)/20)].append((d,x))
    bases=[]
    for _,items in sorted(g.items(),key=lambda kv:-len(kv[1])):
        c=[st.median(i[1][0] for i in items),st.median(i[1][1] for i in items)]
        for b in bases:
            if hav(b['ll'],c)<10000: b['items']+=items;break
        else: bases.append({'ll':c,'items':list(items)})
    bases=[b for b in bases if len(b['items'])>=8]
    M=timedelta(days=45)
    for b in bases:
        ds=sorted(i[0] for i in b['items'])
        b['ll']=[round(st.median(i[1][0] for i in b['items']),6),round(st.median(i[1][1] for i in b['items']),6)]
        b['from'],b['to'],b['n']=ds[0]-M,ds[-1]+M,len(b['items'])
    def act(t): return [i for i,b in enumerate(bases) if b['from']<=t.date()<=b['to']] or list(range(len(bases)))
    def hd(t,x): return min(hav(bases[i]['ll'],x) for i in act(t))
    def nb_(t,x): return min(act(t),key=lambda i:hav(bases[i]['ll'],x))
    dist=[hd(o[0],o[2]) for o in obs]
    sp=[];cur=None
    for i in range(len(obs)):
        if dist[i]>P['NEAR_M']: cur=[i,i] if cur is None else [cur[0],i]
        elif cur: sp.append(tuple(cur));cur=None
    if cur: sp.append(tuple(cur))
    ak=[a['s'] for a in acts]; vk=[v['s'] for v in visits]
    out=[]
    for s,e in sp:
        t0,t1=obs[s][0],obs[e][1]
        dur=(t1-t0).total_seconds()/3600
        vs=[]
        for j in range(bisect.bisect_left(vk,t0),bisect.bisect_right(vk,t1)):
            v=visits[j]
            if v['ll']: vs.append(((v['e']-v['s']).total_seconds()/60, hd(v['s'],v['ll'])/1000))
        as_=[]
        for j in range(bisect.bisect_left(ak,t0),bisect.bisect_right(ak,t1)):
            a=acts[j]
            ends=[hd(a['s'],x)/1000 for x in (a['a'],a['b']) if x]
            as_.append((a['t'],a['d']/1000,max(ends) if ends else 0,(a['e']-a['s']).total_seconds()/60))
        dw=[d for m,d in vs if m>=P['MIN_DWELL_MIN']]
        kinds=collections.Counter(obs[i][3] for i in range(s,e+1))
        # farthest obs, and farthest obs that is corroborated: >FAR for >=2 consecutive obs
        far_idx=[i for i in range(s,e+1) if dist[i]/1000>P['FAR_KM']]
        far_min=(obs[far_idx[-1]][1]-obs[far_idx[0]][0]).total_seconds()/60 if far_idx else 0
        runs=[];cr=None
        for i in range(s,e+1):
            if dist[i]/1000>P['FAR_KM']: cr=[i,i] if cr is None else [cr[0],i]
            elif cr: runs.append(cr);cr=None
        if cr: runs.append(cr)
        runs=[dict(min=(max(obs[i][1] for i in range(a,b+1))-obs[a][0]).total_seconds()/60,n=b-a+1,t=obs[a][0],
                   far=max(dist[a:b+1])/1000,kinds=''.join(sorted(set(obs[i][3] for i in range(a,b+1))))) for a,b in runs]
        out.append(dict(t0=t0,t1=t1,dur=dur,n=e-s+1,vs=vs,acts=as_,runs=runs,
            run_min=max([r['min'] for r in runs],default=0), run_n=max([r['n'] for r in runs],default=0),trunc=(s==0 or e==len(obs)-1),
            maxobs=max(dist[s:e+1])/1000, nfar=len(far_idx), far_min=far_min,
            maxvis=max([d for m,d in vs],default=0), maxdw=max(dw,default=0),
            maxact=max([a[2] for a in as_],default=0),
            fly=[a for a in as_ if a[0] and 'FLY' in a[0].upper()],
            b0=nb_(obs[s-1][0],obs[s-1][2]) if s>0 else None,
            b1=nb_(obs[e+1][0],obs[e+1][2]) if e+1<len(obs) else None,
            kinds=dict(kinds),
            ok=bool(dw) and max(dw)>P['FAR_KM'] and (e-s+1)>=P['MIN_OBS'] and dur>=P['MIN_HRS'],
            gates=(e-s+1)>=P['MIN_OBS'] and dur>=P['MIN_HRS']))
    # minutes of observations near home between each span and the next
    for k,x in enumerate(out):
        if k+1<len(sp):
            a,b=sp[k][1]+1,sp[k+1][0]-1
            x['home_after']=(max(obs[i][1] for i in range(a,b+1))-obs[a][0]).total_seconds()/60
    for k,x in enumerate(out): x['k']=k
    return out,bases,nb

def overlap(a0,a1,b0,b1): return a0<=b1 and b0<=a1

MIN_FAR_MIN=30   # the recommended value (brief, choice B)

def cls(x):
    """same: left and came back to one base. diff: home to home. cut: touches the edge of the file."""
    if x['b0'] is None or x['b1'] is None: return 'cut'
    return 'same' if x['b0']==x['b1'] else 'diff'
MIN_HOME_MIN=30  # as built: a time near home shorter than this is looked through
def chain_cls(sp,x,M):
    """cls() asked of the chain of spans joined by times near home shorter than M minutes."""
    i=j=x['k']
    while i>0 and sp[i-1]['home_after']<M: i-=1
    while j+1<len(sp) and sp[j]['home_after']<M: j+=1
    b0,b1=sp[i]['b0'],sp[j]['b1']
    if b0 is None or b1 is None: return 'cut'
    return 'same' if b0==b1 else 'diff'
def row(label,xs):
    c=collections.Counter(cls(x) for x in xs)
    print(f'   {label:<34} +{len(xs):>3}   same base {c["same"]:>3}   home to home {c["diff"]:>3}   cut {c["cut"]:>3}')
def found(ms,cs): return sum(1 for m in ms if any(overlap(m['s'],m['e'],c['t0'],c['t1']) for c in cs))

if __name__=='__main__':
    args=[a for a in sys.argv[1:] if a!='--list']; LIST='--list' in sys.argv
    for SRC in args:
        visits,acts,paths,mem=load(SRC)
        sp,bases,nb=spans_of(visits,acts,paths)
        cands=[x for x in sp if x['ok']]
        rej=[x for x in sp if not x['ok'] and x['gates']]
        reach=[x for x in rej if x['nfar']]
        c=collections.Counter(cls(x) for x in cands)
        print(f'\n##### {SRC.split("/")[-1]}')
        print(f' today: {len(cands)} candidates, {len(bases)} bases, {nb} outliers dropped; candidates by class: same {c["same"]}, home to home {c["diff"]}, cut {c["cut"]}')
        print(f' rejected spans that pass MIN_OBS and MIN_HRS: {len(rej)}; of those, with an observation beyond FAR_KM: {len(reach)}')
        row('   ... by class',reach)
        print(' no visit of any length beyond FAR_KM:',sum(1 for x in reach if not any(k>P['FAR_KM'] for m,k in x['vs'])),
              '| a visit beyond FAR_KM, but shorter than MIN_DWELL_MIN:',sum(1 for x in reach if any(k>P['FAR_KM'] for m,k in x['vs'])))
        print(' rule 1, lower MIN_DWELL_MIN (a visit beyond FAR_KM of at least D minutes):')
        for D in (45,30,20,10,0): row(f'D = {D}',[x for x in reach if any(m>=D and k>P['FAR_KM'] for m,k in x['vs'])])
        print(' rule 2, a FLYING activity in a span that reaches beyond FAR_KM:')
        row('flying',[x for x in reach if x['fly']])
        print(' rule 3, continuously beyond FAR_KM for at least T minutes (any class):')
        for T in (0,10,20,30,45,60,90,120,180): row(f'T = {T}',[x for x in reach if x['run_min']>=T])
        add=[x for x in reach if cls(x)=='same' and x['run_min']>=MIN_FAR_MIN]
        print(f' recommended in the brief (rule 3, same base only, T = {MIN_FAR_MIN}): {len(cands)} -> {len(cands)+len(add)}')
        far_run=[x for x in reach if x['run_min']>=MIN_FAR_MIN]
        print(' as built: the same-base question looks through a time near home shorter than M minutes:')
        for M in (0,10,15,30,60,120,180,360):
            ch=[x for x in far_run if chain_cls(sp,x,M)=='same']
            g=f", Google's groupings found {found([m for m in mem if m['km'] is not None and m['km']>=P['FAR_KM']],cands+ch)}" if mem else ''
            print(f'   M = {M:<4} {len(cands)} -> {len(cands)+len(ch)}{g}')
        built=[x for x in far_run if chain_cls(sp,x,MIN_HOME_MIN)=='same']
        lost=[x for x in add if x not in built]
        print(f'   at M = {MIN_HOME_MIN}: {len(lost)} of the brief\'s added spans are no longer added; {sum(1 for x in built if x not in add)} are new')
        # Score, from score.go's default weights and anchors, with destination dwell taken as 0.
        cl=lambda v,f:max(0,min(1,v/f))
        def sc(x,km): d=x['dur']/24; return round(100*(.35*cl(km,1000)+.20*cl(x['n']/d,48)+.15*cl(d,7)),1)
        new=sorted(sc(x,max(r['far'] for r in x['runs'] if r['min']>=MIN_FAR_MIN)) for x in add)
        old=sorted(sc(x,x['maxdw']) for x in cands)
        if new: print(f'   scores of the added candidates: {new}')
        if old: print(f'   existing candidates on the same three components, before dwell: median {st.median(old)}')
        big=[m for m in mem if m['km'] is not None and m['km']>=P['FAR_KM']]
        if mem:
            print(f" Google's own trip groupings: {len(mem)}; reaching FAR_KM or more: {len(big)}; found today {found(big,cands)}, with the recommended rule {found(big,cands+add)}")
            missed=[m for m in big if not found([m],cands+add)]
            h2h=sum(1 for m in missed if any(overlap(m['s'],m['e'],x['t0'],x['t1']) and cls(x)=='diff' and x['nfar'] for x in sp))
            print(f'   still not found: {len(missed)}; of those overlapping a home-to-home span that reaches beyond FAR_KM: {h2h}; the rest overlap no such span')
        if LIST:
            for x in add:
                if x not in built: print('   (not added as built, next line)')
                q=[r for r in x['runs'] if r['min']>=MIN_FAR_MIN]
                fv=[(round(m),round(k)) for m,k in x['vs'] if k>P['FAR_KM']]
                print(f'   + {x["t0"].strftime("%Y-%m-%d %H:%M")}  {x["dur"]:6.1f} h  beyond FAR for {max(r["min"] for r in q):4.0f} min  reached {max(r["far"] for r in q):5.0f} km  far visits (min, km) {fv}  modes {sorted(set(a[0] for a in x["acts"] if a[0]))}')
