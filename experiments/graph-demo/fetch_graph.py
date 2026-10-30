#!/usr/bin/env python3
"""Build a graph dataset from a Modrinth .mrpack.

Usage: python3 -I fetch_graph.py <pack.mrpack> <out.json>

Only stdlib. Downloads nothing except Modrinth API metadata.
"""
import json
import os
import re
import sys
import time
import urllib.parse
import urllib.request
import zipfile
from collections import Counter, defaultdict

API = "https://api.modrinth.com/v2"
UA = "modpack-studio-graph-spike/0.1 (local research)"


def get_json(url):
    req = urllib.request.Request(url, headers={"User-Agent": UA})
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.load(resp)


def bulk(endpoint, ids, chunk=50):
    out = []
    ids = list(dict.fromkeys(ids))
    for i in range(0, len(ids), chunk):
        part = ids[i:i + chunk]
        q = urllib.parse.quote(json.dumps(part, separators=(",", ":")), safe="")
        out.extend(get_json(f"{API}/{endpoint}?ids={q}"))
        time.sleep(0.15)
    return out


def parse_file_ids(url):
    m = re.search(r"/data/([^/]+)/versions/([^/]+)/", url or "")
    if m:
        return m.group(1), m.group(2)
    return None, None


def filename_from_url(url):
    if not url:
        return ""
    name = urllib.parse.unquote(url.rsplit("/", 1)[-1])
    return name


# ordered by visual priority: content first, infrastructure last
CATEGORY_ORDER = [
    ("technology", "科技", "#3b82f6"),
    ("magic", "魔法", "#a855f7"),
    ("adventure", "冒险", "#22c55e"),
    ("worldgen", "世界生成", "#14b8a6"),
    ("decoration", "建筑装饰", "#f59e0b"),
    ("building", "建筑", "#f59e0b"),
    ("storage", "存储物流", "#06b6d4"),
    ("transportation", "运输", "#0ea5e9"),
    ("food", "农业食物", "#84cc16"),
    ("agriculture", "农业", "#84cc16"),
    ("equipment", "装备工具", "#f97316"),
    ("combat", "战斗", "#ef4444"),
    ("mobs", "生物", "#ec4899"),
    ("quests", "任务", "#10b981"),
    ("game-mechanics", "机制", "#d946ef"),
    ("management", "管理", "#8b5cf6"),
    ("utility", "工具", "#64748b"),
    ("optimization", "优化", "#eab308"),
    ("library", "前置库", "#94a3b8"),
]
CATEGORY_MAP = {k: (label, color) for k, label, color in CATEGORY_ORDER}
FALLBACK = ("未分类", "#64748b")
MISSING = ("缺失前置", "#dc2626")


def primary_category(cats):
    cats = set(cats or [])
    for key, label, color in CATEGORY_ORDER:
        if key in cats:
            return key, label, color
    return "uncategorized", FALLBACK[0], FALLBACK[1]


def main():
    if len(sys.argv) != 3:
        print(__doc__)
        return 2
    pack_path, out_path = sys.argv[1], sys.argv[2]

    with zipfile.ZipFile(pack_path) as z:
        index = json.loads(z.read("modrinth.index.json"))

    files = index.get("files", [])
    version_ids = []
    for f in files:
        url = (f.get("downloads") or [""])[0]
        _, vid = parse_file_ids(url)
        if vid:
            version_ids.append(vid)
    version_ids = list(dict.fromkeys(version_ids))

    versions = bulk("versions", version_ids)
    vmap = {v["id"]: v for v in versions}

    project_ids = []
    for v in versions:
        if v.get("project_id"):
            project_ids.append(v["project_id"])
    project_ids = list(dict.fromkeys(project_ids))
    projects = bulk("projects", project_ids)
    pmap = {p["id"]: p for p in projects}

    nodes = {}
    file_rows = []
    for f in files:
        url = (f.get("downloads") or [""])[0]
        pid, vid = parse_file_ids(url)
        v = vmap.get(vid)
        if v and v.get("project_id"):
            pid = v["project_id"]
        p = pmap.get(pid) if pid else None
        name = (p or {}).get("title") or (v or {}).get("name") or filename_from_url(url) or f["path"]
        cats = (p or {}).get("categories") or []
        key, label, color = primary_category(cats)
        node_id = f"modrinth:{pid}" if pid else f"file:{f['path']}"
        nodes[node_id] = {
            "id": node_id,
            "label": name,
            "projectId": pid,
            "versionId": vid,
            "version": (v or {}).get("version_number", ""),
            "categories": cats,
            "category": key,
            "categoryLabel": label,
            "color": color,
            "isLibrary": "library" in cats,
            "clientSide": (p or {}).get("client_side"),
            "serverSide": (p or {}).get("server_side"),
            "icon": (p or {}).get("icon_url"),
            "description": (p or {}).get("description", ""),
            "body": (p or {}).get("body", ""),
            "slug": (p or {}).get("slug", ""),
            "loaders": (v or {}).get("loaders", (p or {}).get("loaders", [])),
            "downloads": (p or {}).get("downloads", 0),
            "sourceUrl": f"https://modrinth.com/mod/{(p or {}).get('slug', '')}" if (p or {}).get("slug") else "",
            "path": f["path"],
            "url": url,
            "fileSize": f.get("fileSize", 0),
            "missing": False,
            "inDegree": 0,
            "outDegree": 0,
        }
        file_rows.append(node_id)

    # dependency edges
    edges = []
    edge_keys = set()
    missing_ids = set()
    for node_id, n in list(nodes.items()):
        v = vmap.get(n["versionId"])
        if not v:
            continue
        for dep in v.get("dependencies") or []:
            dtype = dep.get("dependency_type") or "required"
            target_pid = dep.get("project_id")
            target_vid = dep.get("version_id")
            if not target_pid and target_vid and target_vid in vmap:
                target_pid = vmap[target_vid].get("project_id")
            if not target_pid:
                continue
            target_node = f"modrinth:{target_pid}"
            if target_node not in nodes:
                if dtype == "required":
                    missing_ids.add(target_pid)
                    nodes[target_node] = {
                        "id": target_node,
                        "label": f"缺失前置 #{target_pid}",
                        "projectId": target_pid,
                        "versionId": None,
                        "version": "",
                        "categories": [],
                        "category": "missing",
                        "categoryLabel": MISSING[0],
                        "color": MISSING[1],
                        "isLibrary": False,
                        "clientSide": None,
                        "serverSide": None,
                        "icon": None,
                        "description": "",
                        "body": "",
                        "slug": "",
                        "loaders": [],
                        "downloads": 0,
                        "sourceUrl": "",
                        "path": "",
                        "url": "",
                        "fileSize": 0,
                        "missing": True,
                        "inDegree": 0,
                        "outDegree": 0,
                    }
                else:
                    continue
            key = (node_id, target_node, dtype)
            if key in edge_keys:
                continue
            edge_keys.add(key)
            edges.append({"source": node_id, "target": target_node, "type": dtype})

    # name missing nodes if Modrinth knows them
    if missing_ids:
        try:
            mprojects = bulk("projects", list(missing_ids))
            for p in mprojects:
                n = nodes.get(f"modrinth:{p['id']}")
                if n:
                    n["label"] = p.get("title") or n["label"]
                    n["categories"] = p.get("categories") or []
                    n["icon"] = p.get("icon_url")
                    n["description"] = p.get("description", "")
                    n["body"] = p.get("body", "")
                    n["slug"] = p.get("slug", "")
                    n["sourceUrl"] = f"https://modrinth.com/mod/{p.get('slug', '')}" if p.get("slug") else ""
                    n["url"] = f"https://modrinth.com/project/{p.get('slug', p['id'])}"
        except Exception as e:
            print(f"warn: missing project lookup failed: {e}", file=sys.stderr)

    for e in edges:
        if e["source"] in nodes:
            nodes[e["source"]]["outDegree"] += 1
        if e["target"] in nodes:
            nodes[e["target"]]["inDegree"] += 1

    # collapse duplicate file rows for the same project
    seen = set()
    real_nodes = []
    for nid in file_rows:
        if nid in seen:
            continue
        seen.add(nid)
        real_nodes.append(nodes[nid])
    for nid, n in nodes.items():
        if nid not in seen:
            real_nodes.append(n)

    zh_names = {}
    try:
        with open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "zh_names.json"), encoding="utf-8") as fh:
            zh_names = json.load(fh)
    except Exception as e:
        print(f"warn: zh_names.json not loaded: {e}", file=sys.stderr)

    for n in real_nodes:
        n["degree"] = n["inDegree"] + n["outDegree"]
        n["hub"] = n["inDegree"] >= 3 and not n["missing"]
        n["isolated"] = n["degree"] == 0 and not n["missing"]
        n["nameZh"] = zh_names.get(n.get("slug") or n.get("label") or "", "")

    cat_counts = Counter(n["category"] for n in real_nodes)
    categories = []
    for key, label, color in CATEGORY_ORDER:
        if cat_counts.get(key):
            categories.append({"key": key, "label": label, "color": color, "count": cat_counts[key]})
    if cat_counts.get("uncategorized"):
        categories.append({"key": "uncategorized", "label": FALLBACK[0], "color": FALLBACK[1], "count": cat_counts["uncategorized"]})
    if cat_counts.get("missing"):
        categories.append({"key": "missing", "label": MISSING[0], "color": MISSING[1], "count": cat_counts["missing"]})

    edge_counts = Counter(e["type"] for e in edges)
    hubs = sorted([n for n in real_nodes if n["hub"]], key=lambda x: -x["inDegree"])[:12]
    missing = [n for n in real_nodes if n["missing"]]
    isolated = [n for n in real_nodes if n["isolated"]]
    deps = index.get("dependencies", {})

    data = {
        "pack": {
            "name": index.get("name"),
            "version": index.get("versionId"),
            "minecraft": deps.get("minecraft"),
            "loader": "forge" if "forge" in deps else ("neoforge" if "neoforge" in deps else ("fabric" if "fabric-loader" in deps else ("quilt" if "quilt-loader" in deps else "vanilla"))),
            "loaderVersion": deps.get("forge") or deps.get("neoforge") or deps.get("fabric-loader") or deps.get("quilt-loader") or "",
            "source": "modrinth",
            "file": os.path.basename(pack_path),
        },
        "nodes": real_nodes,
        "edges": edges,
        "categories": categories,
        "stats": {
            "files": len(files),
            "nodes": len(real_nodes),
            "edges": len(edges),
            "required": edge_counts.get("required", 0),
            "optional": edge_counts.get("optional", 0),
            "incompatible": edge_counts.get("incompatible", 0),
            "embedded": edge_counts.get("embedded", 0),
            "hubs": len(hubs),
            "isolated": len(isolated),
            "missing": len(missing),
            "libraryMods": sum(1 for n in real_nodes if n["isLibrary"]),
        },
        "hubs": [{"id": n["id"], "label": n["label"], "nameZh": n.get("nameZh", ""), "inDegree": n["inDegree"], "category": n["category"]} for n in hubs],
        "isolatedNodes": [{"id": n["id"], "label": n["label"], "nameZh": n.get("nameZh", ""), "category": n["category"]} for n in isolated],
        "missingNodes": [{"id": n["id"], "label": n["label"], "nameZh": n.get("nameZh", ""), "projectId": n["projectId"]} for n in missing],
        "generatedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    }

    with open(out_path, "w", encoding="utf-8") as fh:
        json.dump(data, fh, ensure_ascii=False, indent=1)

    s = data["stats"]
    print(f"pack: {data['pack']['name']} ({data['pack']['minecraft']} / {data['pack']['loader']} {data['pack']['loaderVersion']})")
    print(f"files={s['files']} nodes={s['nodes']} edges={s['edges']} required={s['required']} optional={s['optional']} embedded={s['embedded']} incompatible={s['incompatible']}")
    print(f"hubs={s['hubs']} isolated={s['isolated']} missing={s['missing']} libraryMods={s['libraryMods']}")
    print("top hubs:", ", ".join(f"{h['label']}({h['inDegree']})" for h in data["hubs"][:8]))
    print("categories:", ", ".join(f"{c['label']}={c['count']}" for c in data["categories"]))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
