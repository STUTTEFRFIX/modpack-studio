# Modpack Studio

> 面向 Minecraft 整合包作者的策展工作台：把模组关系画成一张纸感黑白图，而不是一棵会丢信息的树。

![桌面截图](docs/screenshots/desktop.png)

## 这是什么

Modpack Studio 读取整合包，拆出模组、分类、依赖和联动关系，生成可交互的关系图：

- 节点大小 = 包内被依赖数
- 实线 = 必需前置，虚线 = 可选联动
- 点击节点查看介绍、直接前置、前置的前置、被依赖列表
- 分类各占一块区域，带虚线范围和名称，一眼看出功能分组
- 顶部「图 / 树」切换：图看依赖网络；树是节点-连线图（包 → 分类 → mod），点分类原地展开
- 支持中文译名、搜索、筛选、手机竖屏/横屏、桌面
- 导入后可按文件哈希反查 Modrinth，补分类、图标、简介
- 主页介绍可一键译成中文（MyMemory，按需触发、本地缓存）

当前版本：`0.4.1-tree-svg`（alpha）。

## 快速开始

### 直接看图

打开 `internal/webui/graph.html`，或运行桌面版：

```sh
go run ./cmd/modpack-studio-desktop
```

它会启动本地服务并打开默认浏览器。

### CLI 分析整合包

```sh
go run ./cmd/modpack-studio analyze path/to/pack.mrpack
go run ./cmd/modpack-studio analyze path/to/pack.mrpack --json
```

Go 解析器支持：CurseForge、Modrinth、MCBBS、MultiMC/Prism、HMCL、HMCL Server、GDLauncher、packwiz（基础元数据）、裸 ZIP；FTB/ATLauncher/Technic 有格式识别。

### 导入整合包

在页面顶部点「导入」，选择整合包文件。浏览器内会自动识别两种格式。

**Modrinth `.mrpack`**（无需配置）：

1. JSZip 解压
2. 读取 `modrinth.index.json`
3. 调用 Modrinth API 拉取版本、项目、分类、依赖
4. 在浏览器里生成图数据并保存
5. 重载页面显示

**CurseForge `.zip`**（需要 API Key）：

1. JSZip 解压，读取 `manifest.json`（`manifestType: minecraftModpack`）
2. 用 `x-api-key` 调 `POST /v1/mods` 批量换 mod 名称、分类、简介
3. 逐个查 `GET /v1/mods/{modId}/files/{fileId}` 拿依赖
4. 同样生成图数据并保存

第一次导入 CurseForge 包会弹窗要 Key，存在本机 `localStorage`，不会上传。在 <https://console.curseforge.com> 免费申请；左侧面板有「清除 CurseForge Key」。

CurseForge 的公开 API 允许 `null`（APK 的 `file://`）和 `http://127.0.0.1` 跨域，所以 APK 和网页端都能直连。

**启动器导出 / 带 jar 的 zip**（离线，不需要 Key）：

1. 在包里任意位置找 `mods/**/*.jar`（兼容 Prism/MultiMC、HMCL、PCL2、以及手动打包的实例）
2. 逐个打开 jar，读 `fabric.mod.json`、`quilt.mod.json`、`META-INF/mods.toml`、`META-INF/neoforge.mods.toml`、`mcmod.info`
3. 拿 `id` / `name` / `version` / `depends` / `recommends` / `breaks` 建节点和边
4. `instance.cfg` 的 `name`、`mmc-pack.json` 的组件用来还原整合包名、加载器和 MC 版本

顺序是：`modrinth.index.json` → CurseForge `manifest.json`（且包里没有 jar 时才要 Key）→ `mods/*.jar`。带 jar 的 CurseForge 包会直接走离线解析，不会弹 Key。

### 构建 APK

需要 Android SDK 的 `android.jar`，以及 `aapt2`、`d8`、`apksigner`、`javac`、`keytool`、`zip`。

```sh
ANDROID_JAR=/path/to/android.jar ./scripts/build_apk.sh
```

输出：`android/ModpackStudio-dev.apk`。

### 构建桌面版

```sh
./scripts/build_desktop.sh
```

输出：

- `dist/ModpackStudio.exe`（Windows x64）
- `dist/modpack-studio-desktop-linux`（Linux arm64）

### 重新生成演示数据

```sh
cd experiments/graph-demo
# 需要先准备 packs/createlive5.mrpack
python3 -I fetch_graph.py packs/createlive5.mrpack graph-data.json
python3 -I apply_zh.py graph-data.json zh_names.json
python3 -I build_html.py template.html graph-data.json ../../internal/webui/graph.html
```

示例数据来自 Modrinth 上的 **Create Live 5**，仅用于演示，不随仓库分发 `.mrpack`。

## 目录结构

```text
modpack-studio/
├── cmd/
│   ├── modpack-studio/          # Go CLI：格式识别/清单解析
│   └── modpack-studio-desktop/  # 本地 HTTP 服务 + 打开浏览器
├── internal/
│   ├── model/                   # 跨格式数据模型
│   ├── pack/                    # 格式嗅探与清单解析
│   └── webui/                   # 内嵌的 graph.html
├── android/                     # WebView APK 工程
├── experiments/graph-demo/      # 图数据管线、模板、示例数据
├── docs/                        # 格式文档、计划、交接文档、截图
├── scripts/                     # APK/桌面/演示数据构建脚本
├── go.mod
├── LICENSE
└── THIRD_PARTY.md
```

## 文档

- [交接文档](docs/HANDOVER.md)
- [整合包格式全解](docs/Minecraft整合包格式全解.md)
- [实现计划](docs/整合包工作台实现计划.md)
- [安装与发布说明](docs/INSTALL.md)
- [图数据管线说明](docs/graph-demo.md)
- [文档索引](docs/README.md)

## 当前限制

- 运行时导入支持 Modrinth `.mrpack` 和 CurseForge `.zip`；后者需要 API Key
- CurseForge 依赖要逐个文件查，95 个 mod 大约十几秒
- MCBBS / MultiMC / packwiz 只有 Go 解析器，运行时还不能导入
- Go 核心解析器还没有接到 Web UI；当前 UI 导入由页面内 JS 完成
- 中文译名是人工映射，未知 mod 显示英文
- 不支持编辑、更新、卸载、导出
- APK 使用测试签名；Windows EXE 尚未在 Windows 实机验证
- 示例图数据包含 Modrinth 上公开的模组介绍文本，版权归原作者

## 许可证

MIT。第三方组件与数据来源见 [THIRD_PARTY.md](THIRD_PARTY.md)。
