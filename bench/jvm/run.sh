#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
root=$(cd ../.. && pwd)
komar="$root/bin/komar"
n=${N:-2000000}
reps=${REPS:-3}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

javac --release 8 -Xlint:-options -g -d "$work/classes" Bench.java

best() {
  local min=""
  for _ in $(seq "$reps"); do
    local v
    v=$("$@" 2>/dev/null | awk '/^ns_per_call/ {print $2}')
    if [ -z "$min" ] || awk "BEGIN {exit !($v < $min)}"; then min=$v; fi
  done
  echo "$min"
}

base=$(best java -cp "$work/classes" Bench "$n")
printf '%-22s %10s ns/call\n' "plain java" "$base"
for spec in specs/*.yaml; do
  name=$(basename "$spec" .yaml)
  v=$(best "$komar" run "$spec" -o "$work/$name.jsonl" -- java -cp "$work/classes" Bench "$n")
  hits=$(grep -c '"kind":"hit"' "$work/$name.jsonl" || true)
  size=$(wc -c < "$work/$name.jsonl" | tr -d ' ')
  extra=$(awk "BEGIN {printf \"%.0f\", $v - $base}")
  printf '%-22s %10s ns/call  (+%s ns)  hits=%s  bytes=%s\n' "$name" "$v" "$extra" "$hits" "$size"
done
