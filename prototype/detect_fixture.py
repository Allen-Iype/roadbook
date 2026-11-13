"""End-to-end: read a raw Google Timeline export, emit ranked adventure candidates."""
import json, math, collections, bisect, sys
from datetime import datetime, timedelta
import statistics as st
 
SRC=sys.argv[1] if len(sys.argv)>1 else 'data/fixture-2025-04.segments.timeline.json'
# Where the expected file is written. It holds real coordinates: keep it under data/.
OUT=sys.argv[2] if len(sys.argv)>2 else 'data/fixture-candidates.json'
P=dict(NEAR_M=25000, FAR_KM=100, MIN_OBS=5, MIN_HRS=1.0, MIN_DWELL_MIN=60, MAX_KMH=900, MIN_FAR_MIN=30)
 
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
 
# ---------- parse (Google schema) ----------
raw=json.load(open(SRC))
visits=[];acts=[];paths=[]
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
print(f'parsed {SRC.split("/")[-1]}: {len(visits)} visits, {len(acts)} activities, {len(paths)} path points')
 
# ---------- observations + outlier rejection ----------
obs=[]
for v in visits:
    if v['ll']: obs.append((v['s'],v['e'],v['ll'],True))   # last field: a visit, not a point passed
for a in acts:
    if a['a']: obs.append((a['s'],a['s'],a['a'],False))
    if a['b']: obs.append((a['e'],a['e'],a['b'],False))
for p in paths:
    if p['ll']: obs.append((p['t'],p['t'],p['ll'],False))
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
nb=sum(1 for i in range(len(obs)) if bad(i))
obs=[o for i,o in enumerate(obs) if not bad(i)]
 
# ---------- home bases ----------
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
print(f'outliers dropped {nb} | home bases {len(bases)}: ' + ', '.join(f'({b["ll"][0]:.3f},{b["ll"][1]:.3f}) n={b["n"]}' for b in bases))
def hd(t,x):
    ds=[hav(b['ll'],x) for b in bases if b['from']<=t.date()<=b['to']] or [hav(b['ll'],x) for b in bases]
    return min(ds)
def nb(t,x):
    # index of the nearest base, among the same set hd measures against
    ix=[i for i,b in enumerate(bases) if b['from']<=t.date()<=b['to']] or list(range(len(bases)))
    return min(ix,key=lambda i:hav(bases[i]['ll'],x))
dist=[hd(o[0],o[2]) for o in obs]

# ---------- a span with no dwelt destination (phase 20, issue #8) ----------
# It is still a trip when it leaves and returns to the same base and was
# continuously beyond FAR_KM for MIN_FAR_MIN minutes. Home-to-home spans never
# qualify (the corridor between two homes is far from both), nor do spans cut
# by the edge of the file. Destination: the farthest visit beyond FAR_KM of any
# length; failing that, the farthest observation in a qualifying far run.
def reached(s,e):
    if s==0 or e==len(obs)-1: return None
    if nb(obs[s-1][0],obs[s-1][2])!=nb(obs[e+1][0],obs[e+1][2]): return None
    run=vis=None;i=s
    while i<=e:
        if dist[i]/1000<=P['FAR_KM']: i+=1;continue
        j=i
        while j+1<=e and dist[j+1]/1000>P['FAR_KM']: j+=1
        ok=(max(obs[k][1] for k in range(i,j+1))-obs[i][0]).total_seconds()/60>=P['MIN_FAR_MIN']
        for k in range(i,j+1):
            c=(dist[k],obs[k][2])
            if obs[k][3] and (vis is None or c>vis): vis=c
            if ok and (run is None or c>run): run=c
        i=j+1
    if run is None: return None
    return vis or run
 
# ---------- spans -> candidates ----------
sp=[];cur=None
for i in range(len(obs)):
    if dist[i]>P['NEAR_M']: cur=[i,i] if cur is None else [cur[0],i]
    elif cur: sp.append(tuple(cur));cur=None
if cur: sp.append(tuple(cur))
ak=[a['s'] for a in acts]; vk=[v['s'] for v in visits]
cand=[]
for s,e in sp:
    t0,t1=obs[s][0],obs[e][1]
    dw=[]
    for j in range(bisect.bisect_left(vk,t0),bisect.bisect_right(vk,t1)):
        v=visits[j]
        if v['ll'] and (v['e']-v['s']).total_seconds()/60>=P['MIN_DWELL_MIN']:
            dw.append((hd(v['s'],v['ll']),v['ll']))
    dur=(t1-t0).total_seconds()/3600
    if (e-s+1)<P['MIN_OBS'] or dur<P['MIN_HRS']: continue
    rch=False
    if dw and max(dw)[0]/1000>P['FAR_KM']: dwmax,far=max(dw)
    else:
        r=reached(s,e)
        if r is None: continue
        (dwmax,far),rch=r,True
    km=0;modes=collections.Counter()
    for j in range(bisect.bisect_left(ak,t0),bisect.bisect_right(ak,t1)):
        km+=acts[j]['d'] or 0
        if acts[j]['t']: modes[acts[j]['t']]+=1
    cand.append(dict(start=t0,end=t1,days=round(dur/24,1),dest_km=round(dwmax/1000),
                     track_km=round(km/1000),stops=len(dw),dest=[round(far[0],4),round(far[1],4)],
                     modes=dict(modes.most_common(3)),reached=rch))
for i,c in enumerate(cand): c['repeat']=sum(1 for e in cand[:i] if hav(e['dest'],c['dest'])<60000)
print(f'\n=== {len(cand)} CANDIDATES from the fixture ===')
print(f'{"#":>3} {"start":<11} {"days":>5} {"dest_km":>8} {"track":>6} {"stops":>5} {"rpt":>4} {"reached":>7}  dest')
for i,c in enumerate(cand,1):
    print(f'{i:>3} {str(c["start"])[:10]:<11} {c["days"]:>5} {c["dest_km"]:>8} {c["track_km"]:>6} {c["stops"]:>5} {c["repeat"]:>4} {"yes" if c["reached"] else "":>7}  {c["dest"]}')
json.dump(dict(params=P,bases=[{k:str(v) if k in('from','to') else v for k,v in b.items() if k!='items'} for b in bases],
               candidates=[{k:(str(v) if k in('start','end') else v) for k,v in c.items()} for c in cand]),
          open(OUT,'w'),indent=1)