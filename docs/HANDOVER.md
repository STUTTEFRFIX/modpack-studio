# 交接文档 / Handover

> 版本：0.3.3-zh  
> 目标读者：接手维护、发版或继续开发的人。  
> 先读 README 了解产品，再读本文了解实现与坑。

---

## 1. 项目现状

已完成：

- 纸感黑白关系图 UI（依赖、联动、分类、孤立节点、枢纽）
- 中文译名、搜索、筛选、详情面板
- 桌面 / 手机竖屏 / 手机横屏三套布局
- 浏览器内导入 Modrinth `.mrpack`
- 浏览器内导入 CurseForge `.zip`（需要用户自己的 API Key）
- Android APK（WebView + 系统文件选择器 + 应用私有存储）
- Windows EXE（本地 HTTP 服务 + 默认浏览器）
- Go 版格式识别与清单解析 CLI
- 格式文档、schema、示例、实现计划

未完成：

- CurseForge / MCBBS / MultiMC / packwiz 的运行时导入
- 编辑、更新、卸载、导出
- 导入数据与 Go 核心的统一
- Windows 实机验证
- 正式签名 / 应用商店发布

---

## 2. 架构

```text
┌─────────────────────────────────────────────────────┐
│ 浏览器 / Android WebView / Windows 默认浏览器        │
│                                                     │
│ graph.html（单文件）                                │
│  ├─ 纸感 UI + 力导向图 + 详情/筛选                  │
│  ├─ JSZip：解压 .mrpack                             │
│  ├─ Modrinth API：版本/项目/依赖                     │
│  └─ 导入结果：localStorage 或 AndroidHost 文件       │
└───────────────────────┬─────────────────────────────┘
                        │
        ┌───────────────┴────────────────┐
        │                                │
 Android APK                     Windows / Linux EXE
 MainActivity                    cmd/modpack-studio-desktop
  ├─ WebView                       ├─ net/http
  ├─ 文件选择器                    ├─ embed internal/webui
  ├─ AndroidHost 文件存储          └─ 打开默认浏览器
  └─ WebChromeClient
```

另外还有一套 **Go CLI**：

```text
cmd/modpack-studio
  └─ internal/pack   格式嗅探 + 清单解析
       └─ internal/model
```

注意：**Go CLI 目前没有接到 Web UI 的导入流程**。UI 导入是页面内 JavaScript 实现的；Go 核心是下一步统一的目标。

---

## 3. 两条数据生成路径

### 3.1 离线示例数据（Python）

```text
.mrpack
  → experiments/graph-demo/fetch_graph.py
  → graph-data.json
  → apply_zh.py 补中文名
  → build_html.py 注入 JSZip/中文名/数据
  → internal/webui/graph.html
```

用途：

- 内置演示数据
- 生成截图
- 验证图数据和中文名映射

### 3.2 运行时导入（浏览器 JS）

```text
用户选择 .mrpack
  → JSZip.loadAsync
  → modrinth.index.json
  → api.modrinth.com/v2/versions?ids=...
  → api.modrinth.com/v2/projects?ids=...
  → buildImportedData()
  → localStorage / AndroidHost
  → location.reload()
```

导入生成的数据 schema 与 `graph-data.json` 基本一致，UI 直接复用。

---

## 4. 目录说明

| 路径 | 作用 |
|---|---|
| `cmd/modpack-studio/` | Go CLI：`analyze` / `sniff` |
| `cmd/modpack-studio-desktop/` | 本地 HTTP 服务，内嵌 `internal/webui` |
| `internal/model/` | `Pack` / `Mod` / `Stats` 等统一模型 |
| `internal/pack/` | 各格式嗅探与解析；含合成 ZIP 测试 |
| `internal/webui/graph.html` | 构建好的单文件工作台（由 Python 管线生成） |
| `android/` | WebView APK 工程、Manifest、Java、图标 |
| `experiments/graph-demo/` | 模板、数据管线、JSZip、中文映射、示例数据 |
| `docs/` | 格式文档、计划、安装说明、交接、截图 |
| `scripts/` | APK / 桌面 / 演示数据构建脚本 |

---

## 5. 运行时导入细节

### 5.1 ZIP 与清单

- `graph.html` 内嵌 JSZip 3.10.1（MIT）。
- 读取 `modrinth.index.json`；支持根目录或嵌套一层目录。
- 先找 `modrinth.index.json`；没有就找 `manifest.json`（`manifestType: minecraftModpack`）走 CurseForge；
- CurseForge 清单但包里带 `mods/*.jar` 时，直接走离线 jar 解析，不弹 API Key；
- 两者都没有、或带 jar，就扫 `mods/**/*.jar` 走启动器导出解析（Prism/MultiMC、HMCL、PCL2、手打 zip 都适用）；
- 完全没有可识别内容才报错，错误信息里会列出包的顶层目录。

### 5.1.2 启动器导出 / 裸 zip

- 不限定 base 目录，正则直接匹配 `(^|/)mods/(子目录/)*名字.jar`。
- 每个 jar 用 JSZip 当嵌套 zip 打开，按优先级读：
  `fabric.mod.json` → `quilt.mod.json` → `META-INF/(neoforge.)mods.toml` → `mcmod.info`。
- `mods.toml` 用一个只覆盖 Forge 那套子集的极简 TOML 解析器（`[[mods]]`、`[[dependencies.<id>]]`、三引号字符串）。
- 依赖：`depends`→required、`recommends`→optional、`breaks`→incompatible；`mandatory=false` 归 optional。
- 内置 id（`minecraft`、`java`、`fabricloader`、`quilt_loader`、`forge`、`neoforge` 等）不建边。
- 包名/加载器/版本从 `instance.cfg`、`mmc-pack.json`、`modpack.json`、CF `manifest.json` 里取。
- 实测 100 个 jar 的合成包：Node 下 90 ms（解压 13 ms + 解析 77 ms），手机上会明显更慢，导入时有进度提示。

### 5.1.3 并行扫描与提速

实测（42 MB 包 / 60 个 jar，本机 Node）：

| 做法 | 耗时 |
|---|---|
| JSZip 顺序解压 | 550 ms |
| 主线程 6 路异步并发 | 624 ms（更慢，JS 单线程） |
| 原生 `DecompressionStream` + Blob/Response | 2487 ms（每次调用开销大） |

结论：纯异步并发没有意义，真正提速只能上 Worker。实现方式：

- 主线程把 `ArrayBuffer` 复制一份给每个 worker（按 200 MB 内存预算限制 worker 数，最多 4 个）；
- worker 里直接跑内联的 JSZip（源码用 `document.scripts` 取，不在包里重复存一份），解压 + 解析它分到的那批 jar；
- worker 顺带算 jar 的 SHA-1，供 Modrinth 反查；
- Worker 不可用、包太大或创建失败时自动回退主线程顺序处理。

**APK 的页面地址**：从 `file:///android_asset/graph.html` 改成 `http://127.0.0.1/graph.html`。
`MainActivity.shouldInterceptRequest` 直接返回 assets 里的 HTML，不监听任何端口。
原因：`file://` 是不透明源，Worker 会被拦，`crypto.subtle` 也受限；`127.0.0.1` 同时满足「安全上下文」和「可建 Worker」。
明文只对 `127.0.0.1` 放开，见 `android/res/xml/network_security_config.xml`。

### 5.1.4 Modrinth 反查

- 每个 jar 的 SHA-1 批量 `POST /v2/version_files`（500 个一批）→ project_id；
- `GET /v2/projects?ids=[...]`（100 个一批）取分类、图标、简介、slug、下载量；
- 分类按 `IMPORT_CATEGORY_ORDER` 映射，查不到的留在「未分类」；
- 中文名优先用 jar 内 `assets/<ns>/lang/zh_cn.json` 里的 `itemGroup.<id>` 等键，其次查 `zh_names.json`；
- 全程失败不影响导入，只是没有分类和译名。

### 5.1.5 布局：分类扇区 + 区域圈

- `recomputeCenters()` 把每个分类放到环上的一个扇区，扇区角宽由该分类的半径决定，
  环半径一直放大到所有扇区放得下（`asin(r/R)` 累加 ≤ 1.75π），避免分类互相压；
- 分类半径 `max(48, sqrt(节点数) × 46)`，`simulate()` 里超出半径就把节点拉回来；
- 碰撞分离按 `r1 + r2 + 15`，纵向再多留 11（标签画在圆点下方），跑到收敛为止；
- `renderClusterRegions()` 给每个还有可见节点的分类画虚线圆 + 名称，筛选后自动重画；
- 导入的包没有官方分类色，`finalizePack()` 用一档十级暖灰分配，仍然只有黑白。

**重要坑（已修）**：`onNodeEnter` 会调 `highlight()` 把其它节点压到 `.dim{opacity:.13}`。
鼠标有 `pointerleave` 会复位，但**触屏抬手不触发**，于是 hover 永远挂着，整张图像"空了"——
只有分类标签还看得见（它们在 `clusterLayer`，不在 `.node` 里）。
现在用 `matchMedia('(hover: hover) and (pointer: fine)')` 区分设备，触屏不做 hover 高亮，
拖动时也会清掉残留状态。

### 5.1.6 主页介绍与中文翻译

- 详情面板「主页介绍」渲染 `node.body`（Modrinth 正文，导入时截 8000 字符）；
- 没有正文但有 slug 时，点翻译会先 `GET /v2/project/{slug}` 补拉；
- 翻译走 MyMemory：`GET https://api.mymemory.translated.net/get?q=...&langpair=en|zh-CN`，
  CORS 为 `*`，匿名额度约 5000 字符/天；
- Google Translate / LibreTranslate 在本机网络下不可达，选型时已排除；
- 默认只译「简介 + 正文前 700 字」，按句号/空格切 ≤420 字符一段，段间间隔 700ms；
- 结果按 `modpack-studio:zh:<nodeId>` 存 localStorage，再打开不重复消耗额度；
- 内容是中文时（CJK 占比 > 30%）跳过翻译。

### 5.1.1 CurseForge 导入

- 清单里只有 `projectID` / `fileID`，必须用官方 API 换数据，所以需要用户自己的 API Key。
- `POST /v1/mods`（每批 50 个 `modIds`）拿名称、分类、简介、logo。
- `GET /v1/mods/{modId}/files/{fileId}` 逐个拿依赖，6 个并发；95 个 mod 约十几秒。
- `relationType` 映射：3 required / 2 optional / 5 incompatible / 1 embedded。
- API Key 存在 `localStorage` 的 `modpack-studio:cf-key`，左侧面板可以清除。
- API 对 `Origin: null`（APK 的 `file://`）和 `http://127.0.0.1` 都返回 CORS 头，允许 `x-api-key`、`content-type`。

### 5.2 Modrinth API

- `GET /v2/versions?ids=[...]`，每 50 个一批。
- `GET /v2/projects?ids=[...]`。
- 依赖来自 `version.dependencies`：
  - `required`：画实线；目标不在包内则生成“缺失前置”节点。
  - `optional`：画虚线；目标不在包内则忽略。
  - `incompatible` / `embedded`：如果目标在包内，也画出来。
- CORS：Modrinth 返回 `Access-Control-Allow-Origin: *`；`file://` 和 `http://127.0.0.1` 都能请求。

### 5.3 中文译名

- `experiments/graph-demo/zh_names.json` 保存 slug → 中文。
- `build_html.py` 把它注入为 `window.ZH_NAMES`。
- 运行时按 `project.slug` 查；未知 slug 显示英文。
- 目前只有示例包 95 个 mod 的映射；新包需要继续补。

### 5.4 存储与重载

- 浏览器：`localStorage`，key `modpack-studio:imported`。
- APK：`AndroidHost.saveImported()` 写入 `getFilesDir()/imported.json`，同时也会尝试 localStorage。
- 导入成功后 `location.reload()`；启动时优先读 AndroidHost，再读 localStorage。
- 「返回示例」清除存储并重载。

### 5.5 APK 文件选择器

- `WebChromeClient.onShowFileChooser` → `ACTION_GET_CONTENT`。
- 结果 URI 通过 `ValueCallback<Uri[]>` 交回 WebView。
- JS 的 `<input type="file">` 读取文件；不需要存储权限。
- `file:///android_asset/graph.html` 页面通过 `fetch` 请求 Modrinth HTTPS 接口。

---

## 6. 构建与发版

### 6.1 Go 测试

```sh
go test ./...
go vet ./...
```

### 6.2 生成 Web UI

```sh
cd experiments/graph-demo
python3 -I fetch_graph.py packs/createlive5.mrpack graph-data.json
python3 -I apply_zh.py graph-data.json zh_names.json
python3 -I build_html.py template.html graph-data.json ../../internal/webui/graph.html
```

### 6.3 构建 APK

```sh
ANDROID_JAR=/path/to/android.jar ./scripts/build_apk.sh
```

**关键坑（已修）**：`javac` 会生成 `MainActivity$1.class`、`$2` 等匿名内部类；`d8` 必须传入全部 `.class` 文件，否则 APK 一启动就：

```text
NoClassDefFoundError: Lcom/modpackstudio/app/MainActivity$1;
```

脚本使用：

```sh
d8 ... $(find build/classes -name '*.class')
```

### 6.4 构建桌面版

```sh
./scripts/build_desktop.sh
```

Windows 产物为 console 程序，双击后：

1. 启动 `127.0.0.1:<随机端口>`
2. 打开默认浏览器
3. 控制台保持打开；关闭控制台停止

### 6.5 发版检查

- [ ] `go test ./...`
- [ ] 重新生成 `internal/webui/graph.html`
- [ ] 构建 APK，安装到真机，确认能启动
- [ ] 导入一个 `.mrpack`，确认节点数/边数
- [ ] 确认「返回示例」可用
- [ ] 构建 EXE，至少在 Linux 上跑同一服务代码
- [ ] 计算 sha256
- [ ] 更新 `docs/INSTALL.md` 和 Release Notes

---

## 7. 已知问题与技术债

1. **图构建逻辑重复**：Python `fetch_graph.py` 和 JS `buildImportedData()` 是两份实现。修改依赖/分类规则时必须同步。
2. **Go 核心未接 UI**：`internal/pack` 只服务 CLI；运行时导入走 JS。长期建议把图构建统一到 Go，再由 HTTP API 提供。
3. **格式覆盖**：Modrinth、CurseForge 已接入；MCBBS 需要 `fileApi`，MultiMC/packwiz 需要另外的解析。
4. **中文名覆盖有限**：`zh_names.json` 目前只覆盖示例包；新包大多显示英文。
5. **示例数据版权**：`internal/webui/graph.html` 和 `experiments/graph-demo/graph-data.json` 含 Modrinth 公开介绍文本；如用于正式发布，建议改为可再生成的数据或获得授权。
6. **APK 签名**：测试签名；正式发布需换 release keystore。
7. **Windows EXE**：只在 Linux 上验证了同一服务代码；Windows 实机未测，未签名可能触发 SmartScreen。
8. **WebView 依赖**：APK 使用系统 WebView；极旧设备可能不支持 `DecompressionStream`，但 JSZip 不依赖它。
9. **大包内存**：JSZip 会把 `.mrpack` 读入内存；超大包（数百 MB）可能占用较高。
10. **localStorage 配额**：导入数据会把 mod 介绍截断到 8000 字符；超大包仍可能超限。

---

## 8. 后续路线

优先级建议：

1. CurseForge / MCBBS 导入（至少解析清单并显示模组列表）
2. 把图构建迁到 Go，JS 只负责渲染
3. 编辑：启用/禁用、手工标签、卸载（依赖感知）
4. 更新检查与安全更新
5. 导出 CF / Modrinth / MCBBS
6. 正式签名、CI 构建、GitHub Releases

---

## 9. 第三方与许可

- JSZip 3.10.1，MIT。
- Modrinth API 与公开元数据；模组介绍版权归原作者。
- 示例整合包 Create Live 5 版权归其作者；仓库不包含 `.mrpack`。
- 图标是项目自制。
- 项目代码 MIT，见根目录 `LICENSE`。
