"""Compare two expected files written by prototype/detect_fixture.py, field by field.

    python3 docs/phase-20/measure/diff_expected.py data/archive-candidates.json data/phase-20/archive-candidates.json

Candidates are matched by span (start, end). Prints counts and field names
only; with --list, also the start date, days and distance of each added
candidate (never a coordinate) for the person whose data it is.
"""
import json, sys, collections
args=[a for a in sys.argv[1:] if a!='--list']; LIST='--list' in sys.argv
old,new=(json.load(open(a)) for a in args)
print('params added:',{k:v for k,v in new['params'].items() if k not in old['params']},
      '| changed:',{k:(old['params'][k],v) for k,v in new['params'].items() if k in old['params'] and old['params'][k]!=v})
print('bases identical:',old['bases']==new['bases'])
key=lambda c:(c['start'],c['end'])
o={key(c):c for c in old['candidates']}; n={key(c):c for c in new['candidates']}
added=[c for k,c in n.items() if k not in o]; removed=[k for k in o if k not in n]
changed=collections.Counter(); same=0
for k,c in o.items():
    if k not in n: continue
    diff=[f for f in c if c[f]!=n[k].get(f)]
    if diff: changed[tuple(diff)]+=1
    else: same+=1
print(f'candidates {len(o)} -> {len(n)}: {same} identical in every old field, {sum(changed.values())} changed, {len(added)} added, {len(removed)} removed')
for fields,cnt in changed.items(): print(f'   changed fields {list(fields)}: {cnt}')
print('   existing candidates marked reached:',sum(1 for k in o if k in n and n[k].get('reached')),
      '| added candidates marked reached:',sum(1 for c in added if c.get('reached')),'of',len(added))
print('   order of the existing candidates kept:',[key(c) for c in new['candidates'] if key(c) in o]==[key(c) for c in old['candidates'] if key(c) in n])
if LIST:
    for c in added: print(f'   + {c["start"][:10]}  {c["days"]:>5} d  {c["dest_km"]:>5} km  stops {c["stops"]}  repeat {c["repeat"]}')
