#!/usr/bin/env bash
# Render Psycho's single HTML report for one participant's writing.
#
# Usage:
#   scripts/render-report.sh <participant.txt | dir-of-txt> [output-dir]
#
# Builds the server and renderer, spins up an isolated instance with its own
# config pointing at the participant text, runs POST /analyze-dir, renders
# the single report template, and writes profile-report.html plus the raw
# analysis.json into the output dir. The temp working copy is removed
# afterwards; nothing is left beyond the output dir.
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
input="${1:?usage: render-report.sh <participant.txt | dir-of-txt> [output-dir]}"
outdir="${2:-report-output}"
port="${PSYCHO_PORT:-8081}"

[[ -e "$input" ]] || { echo "error: $input does not exist" >&2; exit 1; }
mkdir -p "$outdir"
outdir="$(cd "$outdir" && pwd)"

work="$(mktemp -d)"
mkdir -p "$work/input"
if [[ -f "$input" ]]; then
  [[ "$input" == *.txt ]] || { echo "error: participant file must be .txt" >&2; exit 1; }
  cp "$input" "$work/input/"
else
  cp "$input"/*.txt "$work/input/"
fi
[[ -n "$(find "$work/input" -maxdepth 1 -name '*.txt' -print -quit)" ]] || {
  echo "error: no .txt files in $input" >&2; exit 1
}

# Isolated run directory: config, dictionary, calibration, and templates all
# resolve relative to the process cwd, so each gets a copy with dir_path
# pointed at the participant text.
mkdir -p "$work/config" "$work/modules/analyze"
cp "$repo/config/calibration.json" "$work/config/" 2>/dev/null || true
sed -e "s|dir_path:.*|dir_path: \"$work/input\"|" \
    -e "s|port: \"8080\"|port: \"$port\"|" \
    "$repo/config/config.yaml" > "$work/config/config.yaml"
cp "$repo/modules/analyze/dictionary.json" "$work/modules/analyze/"
cp -r "$repo/templates" "$work/templates"
cp -r "$repo/assets" "$work/assets"

(cd "$repo" && go build -o "$work/psycho-server" ./cmd/psycho)
(cd "$repo" && go build -o "$work/rendertemplates" ./cmd/rendertemplates)

srv_pid=""
cleanup() {
  [[ -n "$srv_pid" ]] && kill "$srv_pid" 2>/dev/null || true
  rm -rf "$work"
}
trap cleanup EXIT

(cd "$work" && ./psycho-server) >/dev/null 2>&1 &
srv_pid=$!

resp=""
for _ in $(seq 1 60); do
  if resp="$(curl -fsS -X POST "http://localhost:$port/analyze-dir" \
      -H 'Content-Type: application/json' -d '{}' 2>/dev/null)"; then
    break
  fi
  sleep 0.5
done
[[ -n "$resp" ]] || { echo "error: server did not come up on port $port" >&2; exit 1; }

kill "$srv_pid"
wait "$srv_pid" 2>/dev/null || true
srv_pid=""

printf '%s' "$resp" > "$outdir/analysis.json"
(cd "$work" && ./rendertemplates < "$outdir/analysis.json")
mv "$work"/profile-*.html "$outdir/"

echo "rendered into $outdir:"
ls -1 "$outdir"
