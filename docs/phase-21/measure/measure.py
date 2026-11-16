"""Phase 21 measurement (#23): where home evidence comes from, and what each source finds.

Runs the reference detector (prototype/detect_fixture.py) unchanged, swapping
only the list of home bases, and prints counts. No coordinate, date or place
is printed. Run from the repository root:

    python3 docs/phase-21/measure/measure.py data/Timeline.json [more files]

Home evidence compared:
  labels   INFERRED_HOME visits (today's rule)
  HOME     visits typed HOME as well (a type today's rule does not read)
  nights   places where nights were spent: each 22:00-06:00 night, on the
           clock of the timestamp's own offset, goes to the visit covering
           most of it if that is at least 3 hours
"""
import sys, io, json, contextlib, collections, statistics as st
from datetime import datetime, timedelta, time as dtime

PROTO=open('prototype/detect_fixture.py').read()
HEAD,REST=PROTO.split('# ---------- home bases ----------')
BASES_SRC,REST=REST.split('print(f\'outliers dropped')
REST='print(f\'outliers dropped'+REST
REST=REST.split("print(f'\\n=== {len(cand)}")[0]

NIGHT_START,NIGHT_END,MIN_NIGHT_HRS=22,6,3
PAD=timedelta(days=45)

def run(src):
    g={}
    sys_argv=sys.argv; sys.argv=['x',src,'/dev/null']
    with contextlib.redirect_stdout(io.StringIO()): exec(HEAD,g)
    sys.argv=sys_argv
    # Phase 22 added a rule to the detector after these figures were recorded; switch it off
    # so the script still prints what the brief says.
    if 'MAX_STRAY_OBS' in g['P']: g['P']['MAX_STRAY_OBS']=0
    return g

def cluster(g,ev,minn):
    """The prototype's own clustering over (date, [lat, lon]) evidence, with a minimum count."""
    hav=g['hav']; grid=collections.defaultdict(list)
    for d,x in ev: grid[(round(x[0]*20)/20,round(x[1]*20)/20)].append((d,x))
    bases=[]
    for _,items in sorted(grid.items(),key=lambda kv:-len(kv[1])):
        c=[st.median(i[1][0] for i in items),st.median(i[1][1] for i in items)]
        for b in bases:
            if hav(b['ll'],c)<10000: b['items']+=items;break
        else: bases.append({'ll':c,'items':list(items)})
    out=[]
    for b in bases:
        if len(b['items'])<minn: continue
        ds=sorted(i[0] for i in b['items'])
        out.append({'ll':[round(st.median(i[1][0] for i in b['items']),6),round(st.median(i[1][1] for i in b['items']),6)],
                    'from':ds[0]-PAD,'to':ds[-1]+PAD,'n':len(b['items']),'dates':ds})
    return out

def nights(g):
    """One (date, loc) per night held by a visit; plus the count of nights no visit holds."""
    vs=[v for v in g['visits'] if v['ll']]
    if not vs: return [],0
    best={}  # night date -> (hours, loc)
    for v in vs:
        d=v['s'].date()-timedelta(days=1)
        while True:
            a=datetime.combine(d,dtime(NIGHT_START),v['s'].tzinfo); b=a+timedelta(hours=(24-NIGHT_START)+NIGHT_END)
            if a>v['e']: break
            h=(min(b,v['e'])-max(a,v['s'])).total_seconds()/3600
            if h>=MIN_NIGHT_HRS and h>best.get(d,(0,None))[0]: best[d]=(h,v['ll'])
            d+=timedelta(days=1)
    first=min(v['s'] for v in vs).date(); last=max(v['e'] for v in vs).date()
    total=(last-first).days
    return [(d,x) for d,(h,x) in sorted(best.items())],total-len(best)

def detect(g,bases):
    h=dict(g); h['bases']=bases
    with contextlib.redirect_stdout(io.StringIO()): exec(REST,h)
    return h['cand'],h['sp'],h['obs']

def memories(src):
    out=[]
    for s in json.load(open(src))['semanticSegments']:
        if 'timelineMemory' in s:
            m=s['timelineMemory']; m=m.get('trip',m)
            try: km=float(m.get('distanceFromOriginKms'))
            except: km=None
            if km is not None and km>=100: out.append((datetime.fromisoformat(s['startTime']),datetime.fromisoformat(s['endTime'])))
    return out

def line(label,g,bases,mem,ref=None):
    if not bases: print(f'   {label:<46} no home base: no candidates'); return None
    cand,sp,obs=detect(g,bases)
    days=sorted(c['days'] for c in cand)
    found=sum(1 for a,b in mem if any(a<=c['end'] and c['start']<=b for c in cand))
    same='' if ref is None else f"  same span as today {sum(1 for c in cand if (c['start'],c['end']) in ref):>3}"
    gm=f'  Google groupings found {found}/{len(mem)}' if mem else ''
    print(f'   {label:<46} bases {len(bases):>2}  spans {len(sp):>4}  candidates {len(cand):>3}  reached {sum(1 for c in cand if c["reached"]):>2}'
          f'  median {round(st.median(days),1) if days else 0:>4} d  over 7 d {sum(1 for d in days if d>7):>2}  longest {max(days,default=0):>5} d{same}{gm}')
    return cand

def extend(g,today,ng,K,G):
    """Today's bases, plus a period for every run of K or more nights spent at a base's place (nights at most G days apart)."""
    hav=g['hav']; bs=list(today)
    for b in today:
        at=sorted(d for d,x in ng if hav(x,b['ll'])<10000)
        runs=[];cur=[]
        for d in at:
            if cur and (d-cur[-1]).days>G: runs.append(cur);cur=[]
            cur.append(d)
        if cur: runs.append(cur)
        bs+=[{'ll':b['ll'],'from':r[0],'to':r[-1],'n':len(r)} for r in runs if len(r)>=K]
    return bs

if __name__=='__main__':
    # A base given a second period is listed twice; "the same base" must mean the same place.
    REST=REST.replace("return min(ix,key=lambda i:hav(bases[i]['ll'],x))","return tuple(bases[min(ix,key=lambda i:hav(bases[i]['ll'],x))]['ll'])")
    assert 'tuple(bases' in REST
    from datetime import date
    for src in sys.argv[1:]:
        g=run(src); hav=g['hav']; visits=g['visits']; mem=memories(src)
        print(f'\n##### {src.split("/")[-1]}')
        print(' visits by type:',dict(collections.Counter(v['type'] for v in visits)))
        lab=[(v['s'].date(),v['ll']) for v in visits if v['type']=='INFERRED_HOME' and v['ll']]
        labH=[(v['s'].date(),v['ll']) for v in visits if v['type'] in('INFERRED_HOME','HOME') and v['ll']]
        ng,unheld=nights(g)
        today=cluster(g,lab,8)
        near=lambda x,bs: min((hav(b['ll'],x) for b in bs),default=1e12)
        print(f' nights held by a visit: {len(ng)}; nights no visit holds: {max(0,unheld)}')
        at=[(d,x) for d,x in ng if near(x,today)<10000]
        out=[(d,x) for d,x in at if not any(b['from']<=d<=b['to'] and hav(b['ll'],x)<10000 for b in today)]
        print(f' nights at the place of one of today\'s bases: {len(at)}; of those, outside that base\'s period: {len(out)}')
        allc=cluster(g,ng,1)
        for K in (8,14,30):
            cs=[c for c in allc if c['n']>=K]
            print(f' places holding {K} or more nights: {len(cs)}; of those, not one of today\'s bases: {sum(1 for c in cs if near(c["ll"],today)>=10000)}')
        print(' detection with each source of home:')
        ref_c=line('today: INFERRED_HOME visits',g,today,mem) or []
        ref={(c['start'],c['end']) for c in ref_c}
        line('visits typed HOME count too',g,cluster(g,labH,8),mem,ref)
        for K in (14,30): line(f'nights only, places with {K}+ nights',g,cluster(g,ng,K),mem,ref)
        for K in (8,14):
            extra=[c for c in cluster(g,ng,K) if near(c['ll'],today)>=10000]
            line(f'today + other places with {K}+ nights (+{len(extra)})',g,today+extra,mem,ref)
        for K in (3,8,14):
            bs=extend(g,today,ng,K,7)
            line(f'today + periods from runs of {K}+ nights at a base',g,bs,mem,ref)
        line('today\'s bases, each active for the whole file',g,[dict(b,**{'from':date(1,1,1),'to':date(9999,1,1)}) for b in today],mem,ref)
        k=[sum(1 for d,x in out if c['start'].date()<=d<=c['end'].date()) for c in ref_c]
        print(f' today\'s {len(ref_c)} candidates by nights spent at a base\'s place outside its period: 1 or more {sum(1 for x in k if x>=1)}, 3 or more {sum(1 for x in k if x>=3)}, 8 or more {sum(1 for x in k if x>=8)}')
        lng=[(c,x) for c,x in zip(ref_c,k) if c['days']>7]
        print(f' of the {len(lng)} longer than 7 days: {sum(1 for c,x in lng if x>=5)} hold 5 or more such nights')
