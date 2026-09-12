#!/usr/bin/env bash
set -euo pipefail

version=${1:?Usage: scripts/release.sh vX.Y.Z [output-directory]}
if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo 'Expected a version such as v0.1.0' >&2
  exit 2
fi
cd "$(dirname "$0")/.."
out=${2:-dist}
mkdir -p "$out"
out=$(cd "$out" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
export CGO_ENABLED=0
for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  export GOOS=${target%/*} GOARCH=${target#*/}
  name=huddlz
  if [[ $GOOS == windows ]]; then name=huddlz.exe; fi
  dir="$stage/huddlz_${version}_${GOOS}_${GOARCH}"
  mkdir -p "$dir"
  go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=$version" -o "$dir/$name" ./cmd/huddlz
  cp README.md "$dir/README.md"
done
# Normalize archive metadata as well as build paths for repeatable checksums.
python3 - "$stage" "$out" "$version" <<'PY'
import gzip
import hashlib
import pathlib
import sys
import tarfile
import zipfile

stage, out = map(pathlib.Path, sys.argv[1:3])
checksums = []
for directory in sorted(stage.iterdir()):
    windows = '_windows_' in directory.name
    archive = out / (directory.name + ('.zip' if windows else '.tar.gz'))
    if windows:
        with zipfile.ZipFile(archive, 'w', compression=zipfile.ZIP_DEFLATED) as bundle:
            for path in sorted(directory.iterdir()):
                info = zipfile.ZipInfo(path.name, (1980, 1, 1, 0, 0, 0))
                info.compress_type = zipfile.ZIP_DEFLATED
                info.create_system = 3
                info.external_attr = (0o100755 if path.suffix == '.exe' else 0o100644) << 16
                bundle.writestr(info, path.read_bytes())
    else:
        with archive.open('wb') as raw, gzip.GzipFile(filename='', fileobj=raw, mode='wb', mtime=0) as gz:
            with tarfile.open(fileobj=gz, mode='w') as bundle:
                for path in sorted(directory.iterdir()):
                    info = bundle.gettarinfo(str(path), arcname=path.name)
                    info.uid = info.gid = info.mtime = 0
                    info.uname = info.gname = ''
                    info.mode = 0o755 if path.name == 'huddlz' else 0o644
                    with path.open('rb') as source:
                        bundle.addfile(info, source)
    checksums.append(f'{hashlib.sha256(archive.read_bytes()).hexdigest()}  {archive.name}\n')
(out / f'huddlz_{sys.argv[3]}_checksums.txt').write_text(''.join(checksums))
PY
