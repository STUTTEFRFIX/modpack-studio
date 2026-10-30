# 图优先原型（Graph Spike）

> 在仓库中，构建产物是 `internal/webui/graph.html`，截图在 `docs/screenshots/`；本目录下的 `graph.html` 和 `graph-*.png` 是本地生成文件，不提交。

这个目录不是最终实现，而是用来验证一个判断：

> 对整合包作者来说，**依赖/联动关系图**比分类树更有用。

UI 采用**极简黑白纸质文件风格**：米白纸张、墨黑文字、灰阶分类、细线框、点阵网格、衬线标题、等宽元数据；不依赖彩色，重点靠节点大小、线型和标签表达。

## 输入

真实整合包：

- 名称：**Create Live 5**
- 平台：Modrinth
- 版本：`1.3.2.1`
- Minecraft：`1.20.1`
- 加载器：Forge `47.2.17`
- 文件：95 个
- 下载：`packs/createlive5.mrpack`（18 MB）

Modrinth 项目页：<https://modrinth.com/modpack/create-live-5>

## 产物

| 文件 | 说明 |
|---|---|
| `graph.html` | 自包含交互式关系图（极简黑白纸感），可直接用浏览器/文件查看器打开 |
| `graph-data.json` | 图数据：节点、边、分类、统计、介绍正文 |
| `graph-desktop.png` | 桌面端全图截图 |
| `graph-detail.png` | 桌面端 Create 详情截图 |
| `graph-mobile.png` | 手机竖屏全图截图 |
| `graph-mobile-detail.png` | 手机竖屏 Create 详情截图 |
| `graph-landscape.png` | 手机横屏全图截图（1000×500） |
| `graph-landscape-detail.png` | 手机横屏 Create 详情截图 |
| `graph-landscape-800.png` | 小横屏截图（800×400） |
| `zh_names.json` | slug → 中文译名映射 |
| `jszip.min.js` | 浏览器内解压 `.mrpack`（JSZip 3.10.1，MIT） |
| `apply_zh.py` | 把中文译名写进 graph-data.json |
| `fetch_graph.py` | 从 `.mrpack` 生成图数据（会自动读取 zh_names.json） |
| `build_html.py` | 把数据、JSZip、中文名嵌入模板生成 HTML |
| `template.html` | 前端模板（力导向图、筛选、详情、导入） |

## 布局

同一个 `graph.html` 会随方向自动切换：

- **桌面**：左侧筛选、中间图、右侧详情
- **手机竖屏**：图占满舞台，详情从底部弹出
- **手机横屏**：图占满舞台，详情从右侧滑出，筛选从左侧滑出
- 横屏下节点被选中时会自动居中到详情面板左侧的可视区域

## 导入整合包

- 在 `graph.html` / APK / EXE 里点「导入」，选择 Modrinth `.mrpack`
- 浏览器内用 JSZip 读取清单，再查 Modrinth API 获取版本、项目、依赖和介绍
- 生成图数据后写入本地存储并重载；APK 额外写入应用私有文件
- 「返回示例」恢复内置的 Create Live 5 数据
- 当前仅支持 Modrinth `.mrpack`；CurseForge / MCBBS 导入尚未接入
- 中文译名只覆盖 `zh_names.json` 里已有的 slug，未知 mod 显示英文

## 中文译名

- `zh_names.json` 按 slug 保存人工整理的中文译名，可随时修改。
- 图上的可见标签会在英文名下方显示中文；详情标题、依赖列表、枢纽/孤立列表、tooltip、搜索都支持中文。
- 可在左侧「视图」里取消「显示中文译名」。
- 中文名目前是人工映射，未覆盖到的 mod 可以通过 MC百科等来源继续补。

## 复现

```sh
# 1. 下载（如果 packs/ 里还没有）
mkdir -p packs
curl -L -o packs/createlive5.mrpack \
  'https://cdn.modrinth.com/data/VUmd23oG/versions/QKVowWlC/Create%20Live%205-1.3.2.1.mrpack'

# 2. 生成图数据（只访问 Modrinth API，不下载 mod jar）
#    fetch_graph.py 会自动读取 zh_names.json
python3 -I fetch_graph.py packs/createlive5.mrpack graph-data.json

# 如果 graph-data.json 已存在，只补中文名：
python3 -I apply_zh.py graph-data.json zh_names.json

# 3. 生成自包含 HTML
python3 -I build_html.py template.html graph-data.json graph.html
```

## 数据观察

```text
文件 95 / 节点 95 / 关系边 71
必需依赖 51 / 可选联动 20 / 冲突 0 / 内置 0
枢纽 4 / 孤立 27 / 前置库 15 / 缺失前置 0
```

被依赖最多：

| mod | 被依赖数 | 说明 |
|---|---:|---|
| Create | 26 | 既是核心玩法，也是大量 addon 的前置 |
| Just Enough Items (JEI) | 11 | 大量 mod 的可选/必需联动 |
| KubeJS | 5 | 脚本/魔改核心 |
| Architectury API | 3 | 典型前置库 |

这组数据正好说明为什么树不够：

- Create 如果按分类只会落在“科技”，看不出它是 26 个 mod 的枢纽。
- JEI 既是玩法 mod，也是 11 个 mod 的联动对象。
- 20 条 optional 边就是“联动”关系；树会把它们全部丢掉。
- 27 个孤立节点没有依赖边，用图一眼就能看出来。

## 局限

- 目前只解析 Modrinth `.mrpack`；CurseForge 还需要 API Key。
- “联动”目前用 Modrinth 的 `optional` 依赖近似；它不是语义上的“玩法联动”，但已经能揭示大量关系。
- 没有读取本地 jar 的 `mods.toml` / `fabric.mod.json`；所有元数据来自 Modrinth。
- 没有做版本约束检查；只画出了当前安装版本的关系。
- 这是 Python 原型，最终核心会按计划迁到 Go。

## 结论

图优先方向成立。分类树保留为辅助筛选，主视图改为关系图：

- 节点大小 = 包内被依赖数
- 颜色 = 主分类
- 实线 = 必需前置
- 虚线 = 可选联动
- 孤立节点单独标记
- 点击节点看介绍、直接前置、前置的前置、被依赖列表

下一步按 `docs/整合包工作台实现计划.md` 把数据层迁到 Go，并接入 CurseForge / 本地 jar 元数据。
