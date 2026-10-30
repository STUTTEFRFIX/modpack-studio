#!/usr/bin/env python3
"""Attach Chinese display names to graph-data.json.

Usage: python3 -I apply_zh.py graph-data.json zh_names.json
"""
import json
import sys

if len(sys.argv) != 3:
    print(__doc__)
    raise SystemExit(2)

data_path, names_path = sys.argv[1:]
with open(data_path, encoding="utf-8") as f:
    data = json.load(f)
with open(names_path, encoding="utf-8") as f:
    names = json.load(f)

by_id = {n["id"]: n for n in data.get("nodes", [])}
missing = []
for n in data.get("nodes", []):
    key = n.get("slug") or n.get("label") or ""
    n["nameZh"] = names.get(key, "")
    if not n["nameZh"]:
        missing.append(key)

for key in ("hubs", "isolatedNodes", "missingNodes"):
    for item in data.get(key, []):
        node = by_id.get(item.get("id"))
        item["nameZh"] = node.get("nameZh", "") if node else ""

with open(data_path, "w", encoding="utf-8") as f:
    json.dump(data, f, ensure_ascii=False, indent=1)

print(f"attached Chinese names to {len(data.get('nodes', []))} nodes")
if missing:
    print("missing names:", ", ".join(missing))
