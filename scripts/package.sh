#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
bash scripts/build.sh
python3 - <<'PY'
from pathlib import Path
import hashlib,tarfile,zipfile
root=Path('dist'); checks=[]
for directory in sorted(root.glob('openvibecoding-*')):
    if not directory.is_dir(): continue
    if 'windows' in directory.name:
        archive=directory.with_suffix('.zip')
        with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED) as z:
            for name in ['ovc.exe']:z.write(directory/name,name)
    else:
        archive=root/(directory.name+'.tar.gz')
        with tarfile.open(archive,'w:gz') as t:
            for name in ['ovc']:t.add(directory/name,arcname=name)
    checks.append(hashlib.sha256(archive.read_bytes()).hexdigest()+'  '+archive.name)
(root/'SHA256SUMS').write_text('\n'.join(checks)+'\n')
PY
