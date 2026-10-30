#!/usr/bin/env bash
# Regenerate the demo graph data and web UI from a .mrpack.
# Usage: scripts/build_demo.sh [path/to/pack.mrpack]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/experiments/graph-demo"

PACK="${1:-packs/createlive5.mrpack}"
if [ ! -f "$PACK" ]; then
  cat >&2 <<EOF
pack not found: $PACK

Download the demo pack first:
  mkdir -p packs
  curl -L -o packs/createlive5.mrpack \\
    'https://cdn.modrinth.com/data/VUmd23oG/versions/QKVowWlC/Create%20Live%205-1.3.2.1.mrpack'

Then run:
  scripts/build_demo.sh
EOF
  exit 1
fi

python3 -I fetch_graph.py "$PACK" graph-data.json
python3 -I apply_zh.py graph-data.json zh_names.json
python3 -I build_html.py template.html graph-data.json "$ROOT/internal/webui/graph.html"

echo "web UI: $ROOT/internal/webui/graph.html"
echo "data:   $ROOT/experiments/graph-demo/graph-data.json"
