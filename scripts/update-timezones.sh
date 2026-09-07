#!/bin/sh
# Refresh the canonical names used by the backend's Tzdata library.
set -eu
version=${1:-2026c}
case "$version" in
  [0-9][0-9][0-9][0-9][a-z]) ;;
  *) echo 'Usage: sh scripts/update-timezones.sh [IANA release, e.g. 2026c]' >&2; exit 2 ;;
esac
repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM
source_url="https://data.iana.org/time-zones/releases/tzdata${version}.tar.gz"
curl --fail --silent --show-error --location --max-time 30 "$source_url" -o "$temp_dir/tzdata.tar.gz"
# Match Tzdata's source files; aliases are Link records, not Zone records.
tar -xzOf "$temp_dir/tzdata.tar.gz" africa antarctica asia australasia backward etcetera europe northamerica southamerica > "$temp_dir/source"
awk '$1 == "Zone" { print $2 }' "$temp_dir/source" | LC_ALL=C sort -u > "$temp_dir/names"
test -s "$temp_dir/names"
{
  echo "# Canonical Zone names from IANA tzdata ${version} (public domain)."
  echo "# Source: ${source_url}"
  echo '# Regenerate: sh scripts/update-timezones.sh <release>'
  cat "$temp_dir/names"
} > "$repo_dir/internal/cli/timezones.txt"
