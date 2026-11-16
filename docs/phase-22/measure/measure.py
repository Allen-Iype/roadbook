"""Phase 22 measurement: outings cut in two by a short time near home.

Runs the reference detector (prototype/detect_fixture.py). Prints counts and
durations only; no coordinate, date or place. From the repository root:

    python3 docs/phase-22/measure/measure.py data/Timeline.json [more files]

A "gap" is the run of observations near home between two consecutive
away-spans. Its length is from its first observation's start to the latest
end among them, the same measure MIN_HOME_MIN uses.
"""
import sys, io, json, contextlib, collections, statistics as st
from datetime import datetime

PROTO=open('prototype/detect_fixture.py').read()
HEAD,REST=PROTO.split("# ---------- spans -> candidates ----------")
REST=REST.split("print(f'\\n=== {len(cand)}")[0]
# Parsing and bases run once; the distance functions run again for each variant, in the variant's own namespace.
HEAD,FUNCS=HEAD.split("def hd(t,x):")
FUNCS="def hd(t,x):"+FUNCS
# Joining: observations in a gap shorter than JOIN minutes are treated as away, so the two spans become one.
JOINER='''
_sp=[];_cur=None
for _i in range(len(obs)):
    if dist[_i]>P['NEAR_M']: _cur=[_i,_i] if _cur is None else [_cur[0],_i]
    elif _cur: _sp.append(tuple(_cur));_cur=None
if _cur: _sp.append(tuple(_cur))
gaps=[]
for _k in range(len(_sp)-1):
    _a,_b=_sp[_k][1]+1,_sp[_k+1][0]-1
    _m=(max(obs[_i][1] for _i in range(_a,_b+1))-obs[_a][0]).total_seconds()/60
    _br=(obs[_b+1][0]-max(obs[_i][1] for _i in range(_sp[_k][0],_a))).total_seconds()/60   # away before -> away after
    gaps.append((_m,_b-_a+1,any(obs[_i][3] for _i in range(_a,_b+1)),_sp[_k],_sp[_k+1],_br))
    if (_br<BRACKET if BRACKET else _m<JOIN) and (ONLY is None or ONLY(_b-_a+1,any(obs[_i][3] for _i in range(_a,_b+1)))):
        for _i in range(_a,_b+1): dist[_i]=P['NEAR_M']+1
'''

def load(src):
    g={}; av=sys.argv; sys.argv=['x',src,'/dev/null']
    with contextlib.redirect_stdout(io.StringIO()): exec(HEAD,g)
    sys.argv=av
    # The detector now carries this phase's rule. Switch it off, so that "today" below is the
    # detector as it was when the brief was written and the joining is done by this script alone.
    g['P']['MAX_STRAY_OBS']=0
    return g

def detect(g,join,only=None,bracket=None):
    h=dict(g); h['JOIN']=join; h['ONLY']=only; h['BRACKET']=bracket
    with contextlib.redirect_stdout(io.StringIO()): exec(FUNCS+JOINER+REST,h)
    return h['cand'],h['gaps'],h['sp'],h['obs']

def memories(src):
    out=[]
    for s in json.load(open(src))['semanticSegments']:
        if 'timelineMemory' in s:
            m=s['timelineMemory']; m=m.get('trip',m)
            try: km=float(m.get('distanceFromOriginKms'))
            except: km=None
            if km is not None and km>=100: out.append((datetime.fromisoformat(s['startTime']),datetime.fromisoformat(s['endTime'])))
    return out

if __name__=='__main__':
    for src in sys.argv[1:]:
        g=load(src); mem=memories(src)
        cand,gaps,sp,obs=detect(g,0)
        ref={(c['start'],c['end']) for c in cand}
        starts={c['start'] for c in cand}
        is_c=lambda s: obs[s[0]][0] in starts
        print(f'\n##### {src.split("/")[-1]}')
        print(f' today: {len(sp)} spans, {len(cand)} candidates ({sum(1 for c in cand if c["reached"])} reached); gaps between spans: {len(gaps)}')
        print(' gaps by length, and what lies either side (both candidates / one / neither):')
        for lo,hi in ((0,5),(5,15),(15,30),(30,60),(60,120),(120,360),(360,1440)):
            gs=[x for x in gaps if lo<=x[0]<hi]
            both=sum(1 for x in gs if is_c(x[3]) and is_c(x[4])); one=sum(1 for x in gs if is_c(x[3])!=is_c(x[4]))
            print(f'   {lo:>4}-{hi:<4} min: {len(gs):>4} gaps   both {both:>3}   one {one:>3}   neither {len(gs)-both-one:>4}   of one observation {sum(1 for x in gs if x[1]==1):>4}   holding a visit {sum(1 for x in gs if x[2]):>4}')
        def row(label,c2):
            found=sum(1 for a,b in mem if any(a<=c['end'] and c['start']<=b for c in c2))
            gm=f'   Google groupings found {found}/{len(mem)}' if mem else ''
            days=sorted(c['days'] for c in c2)
            print(f'   {label:<44} candidates {len(c2):>3}   reached {sum(1 for c in c2 if c["reached"]):>2}   same span as today {sum(1 for c in c2 if (c["start"],c["end"]) in ref):>3}   longest {max(days,default=0):>5} d{gm}')
        print(' joining spans across a gap shorter than M minutes:')
        for M in (5,15,30,60,120,360):
            row(f'M = {M}',detect(g,M)[0])
        print(' the same, only across gaps that hold no visit:')
        for M in (15,30,60,120):
            row(f'M = {M}, no visit in the gap',detect(g,M,lambda n,v: not v)[0])
        print(' the same, only across gaps of one or two observations:')
        for M in (15,30,60,120):
            row(f'M = {M}, at most two observations',detect(g,M,lambda n,v: n<=2)[0])
        print(' what joining does to today\'s candidates:')
        one_pt=lambda n,v:n==1 and not v
        for J,only,B,label in ((5,lambda n,v:n==1,None,'one observation, any time either side'),(0,one_pt,15,'one point, away within 15 min either side'),(0,one_pt,30,'one point, away within 30 min either side'),(0,one_pt,60,'one point, away within 60 min either side'),(0,lambda n,v:n<=2 and not v,30,'one or two points, away within 30 min')):
            j=detect(g,J,only,B)[0]; add=[]; merged=0; moved=0
            for c in j:
                ov=[x for x in cand if x['start']<=c['end'] and c['start']<=x['end']]
                if not ov: continue
                if len(ov)>1: merged+=1
                h=(c['end']-c['start']).total_seconds()/3600-sum((x['end']-x['start']).total_seconds()/3600 for x in ov)
                if h>0.01: add.append(h)
                if c['dest_km']!=max(x['dest_km'] for x in ov): moved+=1
            add.sort()
            med=round(add[(len(add)-1)//2],1) if add else 0
            print(f'   {label:<46} {len(cand)} -> {len(j)}; made longer {len(add)} (median {med} h; under 1 h {sum(1 for h in add if h<1)}, 1 to 6 h {sum(1 for h in add if 1<=h<6)}, over 6 h {sum(1 for h in add if h>=6)}); two made one {merged}; destination distance changed {moved}')
        print(' gaps of one observation that is not a visit, by the time between the away observations either side:')
        one=[x for x in gaps if x[1]==1 and not x[2]]
        for lo,hi in ((0,5),(5,15),(15,30),(30,60),(60,120),(120,360),(360,10**9)):
            gs=[x for x in one if lo<=x[5]<hi]
            print(f'   {lo:>4}-{"" if hi>10**6 else hi:<4} min: {len(gs):>4}   beside a candidate {sum(1 for x in gs if is_c(x[3]) or is_c(x[4])):>3}')
        print(' joining across such a gap when that time is under B minutes:')
        for B in (5,15,30,60,120,360,10**9):
            j=detect(g,0,lambda n,v:n==1 and not v,B)[0]
            longer=0
            for c in j:
                ov=[x for x in cand if x['start']<=c['end'] and c['start']<=x['end']]
                if ov and (c['end']-c['start']).total_seconds()-sum((x['end']-x['start']).total_seconds() for x in ov)>36: longer+=1
            row(f'B = {"any" if B>10**6 else B} (made longer {longer})',j)
