#!/usr/bin/env python3
"""Embed graph-data.json, JSZip and Chinese names into template.html -> graph.html."""
import json
import os
import sys

if len(sys.argv) != 4:
    print("usage: build_html.py <template.html> <graph-data.json> <out.html>")
    raise SystemExit(2)

tpl_path, data_path, out_path = sys.argv[1:]
base = os.path.dirname(os.path.abspath(tpl_path))
with open(tpl_path, encoding="utf-8") as f:
    tpl = f.read()
with open(data_path, encoding="utf-8") as f:
    data = json.load(f)
with open(os.path.join(base, "jszip.min.js"), encoding="utf-8") as f:
    jszip = f.read()
with open(os.path.join(base, "zh_names.json"), encoding="utf-8") as f:
    zh = json.load(f)

payload = json.dumps(data, ensure_ascii=False, separators=(",", ":")).replace("</", "<\\/")
zh_payload = json.dumps(zh, ensure_ascii=False, separators=(",", ":")).replace("</", "<\\/")
html = (tpl
        .replace("/*__JSZIP__*/", jszip)
        .replace("/*__ZH_NAMES__*/", zh_payload)
        .replace("/*__DATA__*/", payload))
with open(out_path, "w", encoding="utf-8") as f:
    f.write(html)
print(f"wrote {out_path} ({len(html):,} chars, jszip {len(jszip):,})")
