#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
bash scripts/build.sh
# Second artifact: the Windows desktop shell. Windows first in M1 (the Wails
# window is untested on macOS/Linux, and no installer toolchain — NSIS is not
# available on the machines that run this — is wired up yet, so this is a
# portable exe). Cross-buildable from any host: CGO is off.
version="${OPENVIBECODING_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-dev)}"
version="${version#v}"
dest="dist/openvibecoding-desktop-windows-amd64"
mkdir -p "$dest"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags desktop -trimpath -ldflags "-X main.version=$version" -o "$dest/ovc-desktop.exe" ./cmd/ovc
python3 - <<'PY'
from pathlib import Path
import hashlib,tarfile,zipfile
root=Path('dist'); checks=[]
for directory in sorted(root.glob('openvibecoding-*')):
    if not directory.is_dir(): continue
    if 'desktop' in directory.name:
        names=['ovc-desktop.exe']
    elif 'windows' in directory.name:
        names=['ovc.exe']
    else:
        names=['ovc']
    if 'windows' in directory.name:
        archive=directory.with_suffix('.zip')
        with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED) as z:
            for name in names:z.write(directory/name,name)
    else:
        archive=root/(directory.name+'.tar.gz')
        with tarfile.open(archive,'w:gz') as t:
            for name in names:t.add(directory/name,arcname=name)
    checks.append(hashlib.sha256(archive.read_bytes()).hexdigest()+'  '+archive.name)
(root/'SHA256SUMS').write_text('\n'.join(checks)+'\n')
PY
