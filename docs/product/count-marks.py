import re,sys
sec=None; stats={}; order=[]
for line in open(sys.argv[1], encoding='utf-8'):
    m=re.match(r'^## (\d+)\. (.*)',line)
    if m: sec=m.group(2).split(' — ')[0]; order.append(sec); stats[sec]=[0,0,0,0,0]; continue
    if line.startswith('## Totals'): break
    if sec and line.startswith('| ') and not line.startswith('| line') and not line.startswith('| file:line') and not line.startswith('|---'):
        cells=[c.strip() for c in line.strip().strip('|').split('|')]
        if len(cells)<4: continue
        mark=cells[2]
        s=stats[sec]; s[0]+=1
        if 'OK' in mark: s[1]+=1
        if 'TECH' in mark: s[2]+=1
        if 'HOW' in mark: s[3]+=1
        if 'STALE' in mark: s[4]+=1
tot=[0]*5
for k in order:
    print(f"| {k} | " + " | ".join(str(x) for x in stats[k]) + " |")
    tot=[a+b for a,b in zip(tot,stats[k])]
print("| **all** | " + " | ".join(f"**{x}**" for x in tot) + " |")
