# Minecraft Java 版整合包格式全解

> 版本：2026-10-29  
> 用途：启动器 / 整合包工具 / 转换器的格式参考。  
> 范围：Minecraft Java 版（含主流国际格式与国内启动器格式），不含基岩版 Addon。  
> 原则：**识别看内容，不看扩展名**；`.zip` 可能装的是任意一种格式，`.mrpack` 本质也是 ZIP。

---

## 目录

- [0. 先看结论](#0-先看结论)
- [1. 通用模型](#1-通用模型)
- [2. 格式总览](#2-格式总览)
- [3. 格式识别算法](#3-格式识别算法)
- [4. CurseForge `manifest.json`](#4-curseforge-manifestjson)
- [5. Modrinth `.mrpack`](#5-modrinth-mrpack)
- [6. MultiMC / Prism 实例](#6-multimc--prism-实例)
- [7. packwiz](#7-packwiz)
- [8. HMCL 原生整合包](#8-hmcl-原生整合包)
- [9. MCBBS 整合包（`mcbbs.packmeta`）](#9-mcbbs-整合包mcbbs-packmeta)
- [10. HMCL 服务端整合包](#10-hmcl-服务端整合包)
- [11. ATLauncher `instance.json`](#11-atlauncher-instancejson)
- [12. FTB 整合包](#12-ftb-整合包)
- [13. Technic / Solder](#13-technic--solder)
- [14. GDLauncher `.gdlpack`](#14-gdlauncher-gdlpack)
- [15. PCL2 的整合包处理](#15-pcl2-的整合包处理)
- [16. 裸 ZIP / 懒人包 / 启动器包](#16-裸-zip--懒人包--启动器包)
- [17. 其他与旧格式](#17-其他与旧格式)
- [18. 通用安装器设计](#18-通用安装器设计)
- [19. 下载源与 API 速查](#19-下载源与-api-速查)
- [20. 加载器版本规则](#20-加载器版本规则)
- [21. 安全清单](#21-安全清单)
- [22. 格式转换矩阵](#22-格式转换矩阵)
- [23. 启动器实现对照](#23-启动器实现对照)
- [24. 参考来源](#24-参考来源)
- [附录 A：跨格式字段对照](#附录-a跨格式字段对照)
- [附录 B：本目录附带的 schema 与示例](#附录-b本目录附带的-schema-与示例)

---

## 0. 先看结论

1. **不存在统一的“整合包格式”**。实际是几套事实标准并存：
   - **CurseForge `manifest.json`**：只记 CurseForge 的 `projectID/fileID`，安装时再查 API/CDN。
   - **Modrinth `.mrpack`**：直接给出下载 URL + SHA-1/SHA-512，离线信息最完整。
   - **MultiMC / Prism 实例**：直接描述“游戏版本 + 加载器组件 + 游戏目录”，可以完全自包含。
   - **packwiz**：TOML 元数据 + 索引，适合 Git 管理和增量更新。
   - **MCBBS `mcbbs.packmeta`**：国内事实标准，文件可放自己的 `fileApi`，也可引用 CurseForge。
2. **扩展名不可信**：`.zip` 可以是 CurseForge、MultiMC、MCBBS、纯懒人包；`.mrpack` 只是 ZIP。
3. **判断顺序很重要**：`manifest.json` 同时被 CurseForge 和 MCBBS v1 使用，必须用是否存在 `addons` 字段区分。
4. **需要处理三类差异**：
   - 文件来源：远端 ID / 远端 URL / 全部内置。
   - 覆盖层：`overrides/`、`client-overrides/`、`server-overrides/`、`minecraft/`。
   - 侧别：客户端 / 专用服务端 / 可选。
5. **安全底线**：所有来自压缩包的路径都要做 ZIP Slip 校验；所有下载都尽量校验哈希；HTTPS、限制重定向、限制解压体积。

---

## 1. 通用模型

一个整合包在概念上只有三部分：

```text
整合包 = 清单(manifest) + 覆盖文件(overrides) + 远端文件引用(remote refs)
```

安装时统一抽象成：

```text
1. 读清单，得到 Minecraft 版本、加载器类型/版本
2. 准备游戏本体与加载器
3. 按清单下载远端文件（若有）
4. 解压覆盖文件到实例根目录（按覆盖层次）
5. 写入启动器自己的实例元数据（用于更新/修复）
6. 启动游戏
```

### 1.1 实例根目录

大多数格式最终都落到一个“游戏运行目录”：

```text
.minecraft/           或   minecraft/        (MultiMC/Prism 常用)
├── mods/
├── config/
├── defaultconfig/
├── kubejs/
├── resourcepacks/
├── shaderpacks/
├── saves/
├── options.txt
└── ...
```

清单里的相对路径（如 `mods/foo.jar`）默认都相对于这个根目录。

### 1.2 覆盖层

| 层 | 何时应用 | 说明 |
|---|---|---|
| `overrides/` | 客户端 + 服务端 | 基础覆盖，最先解压 |
| `client-overrides/` | 仅客户端 | 后于 `overrides/`，覆盖同名文件 |
| `server-overrides/` | 仅专用服务端 | 后于 `overrides/`，覆盖同名文件 |

不同格式对目录名有硬编码，也有像 CurseForge 那样用 `overrides` 字段指定目录名的。

### 1.3 侧别（side）

| 取值 | 含义 |
|---|---|
| `required` | 两端都需要 |
| `optional` | 可选，安装时询问用户 |
| `unsupported` | 该端不支持，跳过 |
| `client` / `server` / `both` | packwiz、FTB 等格式的另一套表达 |

### 1.4 哈希

| 算法 | 常见用途 |
|---|---|
| SHA-1 | Modrinth 必需；MCBBS 附加文件；HMCL 服务端包 |
| SHA-256 / SHA-512 | Modrinth 必需；packwiz 可选 |
| MD5 | Technic Solder；packwiz 可选 |
| Murmur2 | CurseForge 文件指纹（32 位），GDLauncher 用来反查 CurseForge |

> Murmur2 是 CurseForge 的 `fileFingerprint`，不是文件内容哈希。它只适合“反查是哪个 CurseForge 文件”，不适合完整性校验。

---

## 2. 格式总览

| 格式 | 典型载体 | 标记文件 | MC / 加载器来源 | 文件来源 | 覆盖目录 | 主要使用者 |
|---|---|---|---|---|---|---|
| CurseForge | `.zip` | `manifest.json`（无 `addons`） | `minecraft.version` + `modLoaders` | CurseForge project/file ID | `overrides/`（可配置） | CurseForge App、Prism、HMCL、FCL、PCL2、ATLauncher、BakaXL |
| Modrinth | `.mrpack` / `.zip` | `modrinth.index.json` | `dependencies` | 清单内 URL + SHA-1/SHA-512 | `overrides/`、`client-overrides/`、`server-overrides/` | Modrinth App、Prism、HMCL、FCL、PCL2、ATLauncher |
| MultiMC / Prism | `.zip` / 目录 | `instance.cfg` + `mmc-pack.json` | `components` | 全部内置（可含 `patches`） | `minecraft/` 或 `.minecraft/` | MultiMC、Prism、HMCL/FCL 导入、PCL2、ATLauncher 导出 |
| packwiz | 目录 / Git / ZIP | `pack.toml` | `[versions]` | `index.toml` + `*.pw.toml` 的 URL | 直接写到游戏目录 | packwiz-installer、Prism、ATLauncher |
| HMCL 原生 | `.zip` | `modpack.json` + `minecraft/pack.json` | `modpack.json.gameVersion` / `pack.json` | 全部内置 | `minecraft/` | HMCL、FCL、PCL2 |
| MCBBS | `.zip` | `mcbbs.packmeta`（v2）或带 `addons` 的 `manifest.json`（v1） | `addons` | `fileApi` + `files` / CurseForge ID | `overrides/` | HMCL、FCL、PCL2、BakaXL |
| HMCL 服务端包 | `.zip` | `server-manifest.json` | `addons` | `fileApi` + `files` | `overrides/` | HMCL、FCL |
| ATLauncher | 目录 | `instance.json` | `launcher.loaderVersion` + 版本 JSON | 私有 + 外部清单 | `overridePaths` | ATLauncher |
| FTB Legacy | `.zip` | `minecraft/pack.json` | `pack.json` 内的 Forge 库 | 全部内置 | `minecraft/` | 旧 FTB Launcher、Prism |
| FTB App | 目录 | `instance.json` + `.ftbapp/version.json` | `modLoader` / `targets` | modpacks.ch API | `overrides/` | FTB App、Prism 导入 |
| Technic | `.zip` / Solder | `bin/modpack.jar` 或 `bin/version.json` | jar 内 `version.json` / Solder build | 内置 / Solder URL | 解压到根目录 | Technic Launcher、Prism |
| GDLauncher | `.gdlpack` | `gdlpack.json` | `dependencies` | 哈希反查 Modrinth/CurseForge | `overrides/`、`clientOverrides/`、`serverOverrides/` | GDLauncher |
| PCL2 导出 | `.zip` / `.mrpack` | `modrinth.index.json` | 同 Modrinth | 同 Modrinth | 同 Modrinth | PCL2 |
| 裸 ZIP / 懒人包 | `.zip` | 无 | 目录推断 / 用户选择 | 全部内置 | 根目录或 `.minecraft/` | 任意启动器（手动） |
| BakaXL | — | 无自有格式 | 支持 CurseForge / Modrinth / MCBBS | 同对应格式 | 同对应格式 | BakaXL |

---

## 3. 格式识别算法

### 3.1 先确定“基目录”

很多整合包会把内容再套一层文件夹：

```text
pack.zip
└── MyPack/
    ├── manifest.json
    └── overrides/
```

因此识别时不能只看 ZIP 根目录，要同时看“根目录”和“每个一级目录”。HMCL 的 MultiMC 解析、PCL2 的 `ArchiveBaseFolder` 都这么做。

建议：

```text
candidate_bases = [""] + [d + "/" for d in 一级目录]
```

但只接受“在 base 下找到标记文件”的候选，不要把任意深层目录都当根。

### 3.2 判定顺序

推荐顺序（与 HMCL / FCL / PCL2 的实际行为一致，差异见注释）：

```text
1. mcbbs.packmeta                          -> MCBBS v2
2. mmc-pack.json + instance.cfg            -> MultiMC / Prism
3. modrinth.index.json                     -> Modrinth
4. manifest.json:
     含 addons 字段                         -> MCBBS v1
     manifestType == "minecraftModpack"     -> CurseForge
5. modpack.json + minecraft/pack.json      -> HMCL 原生
6. server-manifest.json                    -> HMCL 服务端包
7. pack.toml                               -> packwiz
8. gdlpack.json                            -> GDLauncher
9. instance.json:
     有 uuid/id/versionId/mcVersion/modLoader -> FTB App
     有 launcher 对象 + 版本 JSON 字段        -> ATLauncher
10. bin/modpack.jar 或 bin/version.json    -> Technic
11. modpack.zip 或 modpack.mrpack 在包内   -> 启动器包（懒人包）
12. 以上都没有                            -> 裸 ZIP，按目录结构推断
```

**为什么 MCBBS 要在 CurseForge 前**：MCBBS v1 的 `manifest.json` 也写 `manifestType: "minecraftModpack"`，只能靠 `addons` 区分。HMCL 的做法是先用 MCBBS 解析，校验失败（`addons == null`）再落到 CurseForge。

**为什么 MultiMC 要在 `manifest.json` 前**：有些 Prism 导出会同时带一个 CurseForge 兼容清单，但真正的主格式是 `instance.cfg` + `mmc-pack.json`（PCL2 #4194）。

**为什么 Modrinth 要在 `manifest.json` 前**：避免被误判成 CurseForge；两者标记文件不同，实际很少冲突，但顺序仍应固定。

### 3.3 参考伪代码

```python
def sniff(zip_path):
    with ZipFile(zip_path) as z:
        names = set(z.namelist())

        for base in candidate_bases(names):
            def has(n):
                return (base + n) in names

            if has("mcbbs.packmeta"):
                return "mcbbs-v2", base

            if has("mmc-pack.json") and has("instance.cfg"):
                return "multimc", base

            if has("modrinth.index.json"):
                return "modrinth", base

            if has("manifest.json"):
                try:
                    j = json.loads(z.read(base + "manifest.json"))
                except Exception:
                    j = {}
                if "addons" in j:
                    return "mcbbs-v1", base
                if j.get("manifestType") == "minecraftModpack":
                    return "curseforge", base

            if has("modpack.json"):
                if has("minecraft/pack.json"):
                    return "hmcl-native", base
                # 否则可能是已安装实例的配置，不是整合包

            if has("server-manifest.json"):
                return "hmcl-server", base

            if has("pack.toml"):
                return "packwiz", base

            if has("gdlpack.json"):
                return "gdlpack", base

            if has("instance.json"):
                j = read_json(z, base + "instance.json")
                if {"uuid", "versionId", "mcVersion"} <= set(j):
                    return "ftb-app", base
                if "launcher" in j:
                    return "atlauncher", base

            if has("bin/modpack.jar") or has("bin/version.json"):
                return "technic", base

            if has("modpack.zip") or has("modpack.mrpack"):
                return "launcher-bundle", base

        # 兜底：看有没有 .minecraft/、versions/、mods/
        if any(n.startswith(".minecraft/") for n in names):
            return "plain-zip", ".minecraft/"
        if any(n.startswith("minecraft/") for n in names):
            return "plain-zip", "minecraft/"
        if any(n.startswith("mods/") for n in names):
            return "plain-zip", ""
        return "unknown", ""
```

### 3.4 扩展名与 MIME

| 扩展名 | 实际 |
|---|---|
| `.zip` | 任意格式，必须嗅探 |
| `.mrpack` | ZIP；应有 `modrinth.index.json` |
| `.gdlpack` | ZIP；应有 `gdlpack.json` |
| `.hmdp` | 某些 HMCL 版本导出的扩展名；本质是 ZIP，按内容判断 |
| `.mcpack` / `.mcaddon` / `.mcworld` | 基岩版，不是 Java 整合包 |

`.mrpack` 的 MIME 是 `application/x-modrinth-modpack+zip`，但不要依赖 MIME。

---

## 4. CurseForge `manifest.json`

### 4.1 结构

```text
pack.zip
├── manifest.json
├── modlist.html          (可选，人读)
└── overrides/            (目录名由 manifest.overrides 指定)
    ├── config/
    ├── kubejs/
    ├── options.txt
    └── mods/             (仅放 CurseForge 上没有的 jar)
```

### 4.2 字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `manifestType` | string | 是 | 固定 `"minecraftModpack"` |
| `manifestVersion` | int | 是 | 目前为 `1` |
| `name` | string | 是 | 整合包名 |
| `version` | string | 是 | 整合包版本 |
| `author` | string | 是 | 作者 |
| `description` | string | 否 | 描述（部分导出器会写） |
| `minecraft` | object | 是 | 见下 |
| `files` | array | 是 | 远端文件列表，可为空 |
| `overrides` | string | 否 | 覆盖目录名，默认 `"overrides"` |
| `projectID` | int | 否 | CurseForge 项目 ID（发布信息） |
| `fileID` | int | 否 | CurseForge 文件 ID（发布信息） |
| `websiteUrl` | string | 否 | 项目网页 |
| `curse` | string | 否 | 旧字段，部分导出器保留 |

`minecraft`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `version` | string | 如 `"1.20.1"` |
| `modLoaders` | array | `{ "id": "forge-47.2.0", "primary": true }` |

`files[]`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `projectID` | int | CurseForge 项目 ID |
| `fileID` | int | CurseForge 文件 ID |
| `required` | bool | 是否必需；缺失时不同实现行为不一致，**务必显式写** |
| `fileName` | string | 非标准扩展；HMCL 等用来拼 CDN URL |
| `url` | string | 非标准扩展；显式下载地址 |
| `hashes` | object | 非标准扩展；HMCL 支持 `sha1/sha256/sha512/md5` |

> 标准 CurseForge 清单只保证 `projectID` / `fileID` / `required`。`fileName`、`url`、`hashes` 是 HMCL 一类启动器的扩展字段，不能假设所有启动器都认。

### 4.3 示例

```json
{
  "manifestType": "minecraftModpack",
  "manifestVersion": 1,
  "name": "Example Pack",
  "version": "1.2.0",
  "author": "ExampleAuthor",
  "description": "A small example pack.",
  "minecraft": {
    "version": "1.20.1",
    "modLoaders": [
      { "id": "forge-47.2.0", "primary": true }
    ]
  },
  "files": [
    { "projectID": 238222, "fileID": 4593547, "required": true },
    { "projectID": 306612, "fileID": 4311849, "required": false }
  ],
  "overrides": "overrides"
}
```

### 4.4 加载器 ID

`modLoaders[].id` 的格式是 `<loader>-<version>`：

| 加载器 | 示例 | 说明 |
|---|---|---|
| Forge | `forge-47.2.0` | 现代写法 |
| Forge | `forge-1.12.2-14.23.5.2860` | 旧包可能带 MC 前缀 |
| Fabric | `fabric-0.15.11` | loader 版本 |
| Quilt | `quilt-0.23.1` | loader 版本 |
| NeoForge | `neoforge-20.4.237` | 常见写法 |
| NeoForge | `neoforge-1.20.1-47.1.106` | 部分 CurseForge 导出要求带 MC 前缀 |
| LiteLoader | `liteloader-1.12.2` | 罕见 |

实现建议：

```text
1. 取 primary == true 的项；没有则取第一项。
2. 按第一个 '-' 切出 loader 名，剩余部分作为版本。
3. 对 Forge/NeoForge，如果版本形如 "<mc>-<loader>"，去掉 "<mc>-" 前缀再交给加载器安装器。
4. 不认识 loader 时不要静默忽略，至少提示用户。
```

### 4.5 文件解析

清单本身**没有文件名、没有哈希、没有下载 URL**。需要：

1. 调 CurseForge API：
   - `GET https://api.curseforge.com/v1/mods/{projectID}/files/{fileID}`
   - 或 `POST https://api.curseforge.com/v1/mods/files`，body `{"fileIds":[...]}`（批量）
   - Header：`x-api-key: <API_KEY>`
2. 取 `data.fileName`、`data.downloadUrl`、`data.hashes`。
3. 如果 `downloadUrl == null`（作者关闭了第三方分发），只能提示用户，或尝试 CDN 直链。
4. CDN 直链（已知文件名时）：

```text
https://edge.forgecdn.net/files/{fileID / 1000}/{fileID % 1000}/{fileName}
```

也可以使用 `mediafilez.forgecdn.net` 镜像。`/1000` 和 `%1000` 都是整数除法/取余，不是字符串截断。

> 失败常见原因：`fileID` 已删除、文件被作者设为私有、`fileName` 与 CDN 上的实际文件名不一致、API Key 配额用尽。

### 4.6 覆盖层

- `overrides/` 下的内容**直接解压到实例根目录**，不是解压到 `overrides/` 下。
- `overrides/mods/` 里的 jar 是“CurseForge 上没有的 mod”，不需要再下载。
- 可包含 `config/`、`kubejs/`、`defaultconfig/`、`resourcepacks/`、`shaderpacks/`、`options.txt`、存档等。
- `overrides` 字段可以是别的目录名，甚至带路径；实现要使用该字段，而不是硬编码。

### 4.7 可选文件

`required: false` 表示可选。建议安装前让用户勾选，并记录被排除的 `projectID:fileID`，更新时不要重新装回来。

### 4.8 服务端包

CurseForge 的“server pack”通常还是同一套 `manifest.json` + `overrides/`，只是覆盖内容换成服务端配置。没有独立格式。

### 4.9 坑

- `manifest.json` 可能带 UTF-8 BOM；解析前去掉。
- `files` 可能为空，全部内容在 `overrides/`。
- 同一个 `projectID` 可能出现多个 `fileID`（不同版本），按 `fileID` 去重，不要按项目去重。
- 有些包会在 `overrides/mods/` 放一个和远端文件同名的 jar；解压覆盖顺序要明确（通常是先下载远端，再解压覆盖，或相反）。推荐：**先解压 overrides，再下载远端文件**，让远端文件胜出；若希望 overrides 胜出，则反过来。HMCL 的 Curse 安装任务对 `overrides/mods/` 有专门处理，避免重复。
- `required` 缺失时，HMCL 的 primitive boolean 会变成 `false`（视为可选）；其他实现可能默认 `true`。**导出时永远显式写**。

---

## 5. Modrinth `.mrpack`

### 5.1 结构

```text
pack.mrpack
├── modrinth.index.json
├── overrides/
│   ├── config/
│   └── options.txt
├── client-overrides/          (可选)
└── server-overrides/          (可选)
```

`.mrpack` 是 ZIP，`modrinth.index.json` 必须在根目录（部分启动器容忍一层目录，但规范要求根）。

### 5.2 `modrinth.index.json`

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `formatVersion` | number | 是 | 当前 `1` |
| `game` | string | 是 | 只能是 `"minecraft"` |
| `versionId` | string | 是 | 该整合包版本的唯一 ID |
| `name` | string | 是 | 显示名 |
| `summary` | string | 否 | 简介 |
| `files` | array | 是 | 下载文件列表 |
| `dependencies` | object | 是 | 游戏/加载器版本 |

`files[]`：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `path` | string | 是 | 相对实例根目录，如 `mods/foo.jar` |
| `hashes` | object | 是 | 必须含 `sha1` 和 `sha512` |
| `env` | object | 否 | `{ "client": ..., "server": ... }` |
| `downloads` | string[] | 是 | HTTPS URL，按顺序尝试 |
| `fileSize` | int | 是 | 字节数，仅用于进度显示 |

`env` 取值：`required` / `optional` / `unsupported`。缺省视为两端 `required`。

`dependencies` 已知键：

```text
minecraft
forge
neoforge
fabric-loader
quilt-loader
```

值为版本字符串。规范要求忽略不认识的键（未来可能新增），不要因为未知键直接失败。

### 5.3 示例

```json
{
  "formatVersion": 1,
  "game": "minecraft",
  "versionId": "1.0.0",
  "name": "Example Fabric Pack",
  "summary": "A small example pack.",
  "files": [
    {
      "path": "mods/fabric-api-0.92.2+1.20.1.jar",
      "hashes": {
        "sha1": "0000000000000000000000000000000000000000",
        "sha512": "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
      },
      "env": {
        "client": "required",
        "server": "required"
      },
      "downloads": [
        "https://cdn.modrinth.com/data/P7dR8mSH/versions/abc123/fabric-api-0.92.2%2B1.20.1.jar"
      ],
      "fileSize": 1234567
    }
  ],
  "dependencies": {
    "minecraft": "1.20.1",
    "fabric-loader": "0.15.11"
  }
}
```

### 5.4 路径与哈希规则

- `path` 必须是相对路径，禁止：
  - `..` 路径段
  - 以 `/` 或 `\` 开头
  - Windows 盘符（`C:`）
  - 反斜杠
- 规范要求安装器拒绝逃逸实例目录的路径。
- `hashes` **必须**同时有 `sha1` 和 `sha512`；其他算法可忽略。
- 下载 URL 必须是 HTTPS、符合 RFC 3986；Modrinth 上传时只允许：
  - `cdn.modrinth.com`
  - `github.com`
  - `raw.githubusercontent.com`
  - `gitlab.com`
- 必须跟随 3xx 重定向（建议至少 3 次）。
- `fileSize` 只用于进度，**不要**拿它做严格校验（PCL2 明确注释了“Modrinth 整合包不应校验文件大小”）。

### 5.5 覆盖层顺序

规范明确：先 `overrides/`，再 `server-overrides/`（服务端）。客户端同理是 `overrides/` → `client-overrides/`。后应用的覆盖同名文件。

### 5.6 与 CurseForge 的关键差异

| 点 | CurseForge | Modrinth |
|---|---|---|
| 文件定位 | projectID/fileID | 直接 URL |
| 哈希 | 清单里没有 | sha1 + sha512 必需 |
| 离线可装 | 否（要 API/CDN） | 基本可以（URL 仍可能失效） |
| 可选文件 | `required: false` | `env.optional` |
| 侧别 | 无 | `env.client/server` |
| 加载器字段 | `modLoaders[].id` | `dependencies` 键 |

### 5.7 Modrinth API

```text
GET https://api.modrinth.com/v2/project/{id|slug}
GET https://api.modrinth.com/v2/project/{id|slug}/version
GET https://api.modrinth.com/v2/version/{versionId}
GET https://api.modrinth.com/v2/version_file/{hash}?algorithm=sha1
GET https://api.modrinth.com/v2/version_file/{hash}?algorithm=sha512
```

- 建议带 `User-Agent: <app>/<version> (contact)`。
- `version.files[]` 含 `url`、`hashes`、`fileSize`、`filename`。
- `version.dependencies[]` 含 `project_id`、`version_id`、`dependency_type`（`required/optional/incompatible/embedded`）。

---

## 6. MultiMC / Prism 实例

### 6.1 目录结构

```text
instances/MyPack/
├── instance.cfg
├── mmc-pack.json
├── patches/                 (可选，自定义组件补丁)
│   └── com.example.mycomponent.json
├── .minecraft/              (MultiMC 传统)
│   └── ...
├── minecraft/               (Prism 也支持；两者都有时 Prism 优先 minecraft/)
│   └── ...
├── libraries/               (实例私有库，可选)
├── jarmods/                 (jar mod，可选)
├── icon.png                 (可选)
└── .packignore              (导出时排除规则，可选)
```

导出 ZIP 时，通常就是把这个实例目录打包；可能再套一层目录。

### 6.2 `instance.cfg`

INI，`[General]` 段。常见键：

| 键 | 说明 |
|---|---|
| `InstanceType` | 现代实例为 `OneSix`；旧实例为 `Legacy` |
| `ConfigVersion` | 配置版本，常见 `1.2` |
| `name` | 实例显示名 |
| `iconKey` | 图标键，对应 `icon.png` 或内置图标 |
| `notes` | 备注 |
| `OverrideMemory` | 是否覆盖全局内存设置 |
| `MinMemAlloc` / `MaxMemAlloc` | 最小/最大内存（MB） |
| `OverrideJavaArgs` | 是否覆盖 JVM 参数 |
| `JvmArgs` | JVM 参数 |
| `OverrideJavaLocation` | 是否覆盖 Java 路径 |
| `JavaPath` | Java 可执行文件路径 |
| `LaunchMaximized` | 是否最大化 |
| `MinecraftWinWidth` / `MinecraftWinHeight` | 窗口尺寸 |
| `JoinServerOnLaunch` / `JoinServerOnLaunchAddress` | 启动后自动进服 |
| `PreLaunchCommand` / `PostExitCommand` | 启动前/退出后命令 |
| `ManagedPack` | 是否由平台托管 |
| `ManagedPackType` | `modrinth` / `flame` / `technic` / `ftb` 等 |
| `ManagedPackID` | 平台项目 ID |
| `ManagedPackVersionID` | 平台版本 ID |

示例：

```ini
[General]
InstanceType=OneSix
ConfigVersion=1.2
name=Example Pack
iconKey=default
OverrideMemory=true
MinMemAlloc=1024
MaxMemAlloc=4096
OverrideJavaArgs=false
JvmArgs=
OverrideJavaLocation=false
JavaPath=
LaunchMaximized=false
MinecraftWinWidth=854
MinecraftWinHeight=480
```

### 6.3 `mmc-pack.json`

```json
{
  "formatVersion": 1,
  "components": [
    { "uid": "net.minecraft", "version": "1.20.1", "important": true },
    { "uid": "net.minecraftforge", "version": "47.2.0" }
  ]
}
```

组件字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `uid` | string | 组件唯一 ID |
| `version` | string | 组件版本；有的组件由 patch 提供版本，可为空 |
| `important` | bool | 重要组件，用户不能移除 |
| `dependencyOnly` | bool | 仅作为依赖，不在 UI 显示 |
| `cachedName` | string | 缓存的显示名 |
| `cachedVersion` | string | 缓存的版本 |
| `cachedRequires` | array | `{ "uid": ..., "equals": ..., "suggests": ... }` |

常见 UID：

| UID | 含义 |
|---|---|
| `net.minecraft` | Minecraft 本体 |
| `net.minecraftforge` | Forge |
| `net.neoforged` | NeoForge |
| `net.fabricmc.fabric-loader` | Fabric Loader |
| `net.fabricmc.intermediary` | Fabric intermediary |
| `org.quiltmc.quilt-loader` | Quilt Loader |
| `org.quiltmc.hashed` | Quilt hashed 映射 |
| `com.mumfrey.liteloader` | LiteLoader |
| `org.multimc.custom.fakeonline` | 离线上线模式补丁 |
| `com.unascribed.unsup` / `com.unascribed.sup` | unsup 更新器组件 |
| `org.lwjgl*` | LWJGL 相关（导入时通常跳过） |

### 6.4 `patches/`

文件名通常是 `<uid>.json`，内容类似 Mojang 版本 JSON，但不是直接兼容：

```json
{
  "uid": "com.example.mycomponent",
  "name": "My Component",
  "version": "1.0.0",
  "requires": [
    { "uid": "net.minecraft", "equals": "1.20.1" }
  ],
  "mainClass": "...",
  "+tweakers": ["..."],
  "libraries": [
    { "name": "com.example:example:1.0.0", "url": "https://example.com/maven/" }
  ]
}
```

字段支持 `+`/`-` 前缀表示追加/删除（如 `+tweakers`、`+traits`），否则覆盖。常见 traits：`legacyFML`、`legacyLaunch`、`alphaLaunch`、`noapplet`、`FirstThreadOnMacOS`。

### 6.5 导入/导出注意

- ZIP 可能是根目录直接放 `instance.cfg`，也可能套一层；两种都要支持。
- 游戏目录可能是 `.minecraft/` 或 `minecraft/`；**两者都有时 Prism 优先 `minecraft/`**（PCL2 #9272）。
- `patches/` 里的组件必须在 `mmc-pack.json` 里有对应 `uid`，否则不会生效。
- `important` 组件缺失会导致实例无法启动。
- 导出时 `.packignore` 可以排除 `saves/`、`logs/`、`crash-reports/` 等。
- Prism 的托管包键（`ManagedPack*`）用于自动更新；导入第三方包时不要伪造。

---

## 7. packwiz

packwiz 不是“一个压缩包格式”，而是一套**TOML 元数据 + 索引**，通常放在 Git 仓库里，由 `packwiz-installer` 在启动前同步。

### 7.1 目录结构

```text
pack/
├── pack.toml
├── index.toml
├── mods/
│   ├── jei.pw.toml
│   └── (可选) jei.jar
├── config/
├── kubejs/
└── ...
```

### 7.2 `pack.toml`

```toml
name = "Example Pack"
author = "ExampleAuthor"
version = "1.0.0"
pack-format = "packwiz:1.1.0"

[index]
file = "index.toml"
hash-format = "sha256"
hash = "0000000000000000000000000000000000000000000000000000000000000000"

[versions]
minecraft = "1.20.1"
forge = "47.2.0"
# fabric = "0.15.11"
# quilt = "0.23.1"
# liteloader = "1.12.2"
```

字段：

| 字段 | 必填 | 说明 |
|---|---|---|
| `name` | 是 | 显示名 |
| `pack-format` | 是 | `"packwiz:1.1.0"`；省略时按 `1.0.0` |
| `index` | 是 | `{ file, hash-format, hash }` |
| `versions` | 是 | `minecraft` 必填；加载器可选 |
| `author` | 否 | 导出 CurseForge 时使用 |
| `description` | 否 | 导出 Modrinth 时使用 |
| `version` | 否 | 整合包版本，不用于更新检测 |

`index.hash-format` 支持：`sha256`、`sha512`、`sha1`、`md5`、`murmur2`。

### 7.3 `index.toml`

```toml
hash-format = "sha256"

[[files]]
file = "mods/jei.pw.toml"
hash = "0000000000000000000000000000000000000000000000000000000000000000"
metafile = true

[[files]]
file = "config/example.cfg"
hash = "1111111111111111111111111111111111111111111111111111111111111111"
preserve = true
```

字段：

| 字段 | 必填 | 说明 |
|---|---|---|
| `file` | 是 | 相对索引文件的路径，禁止越出包根 |
| `hash` | 是 | 文件哈希 |
| `hash-format` | 否 | 覆盖全局算法 |
| `metafile` | 否 | 该文件是 `.pw.toml` 元文件 |
| `alias` | 否 | 备用下载名（与 metafile 不兼容） |
| `preserve` | 否 | 已存在则保留，不覆盖用户修改 |

### 7.4 `mods/*.pw.toml`

```toml
name = "JEI"
filename = "mods/jei-1.20.1-forge-15.2.0.27.jar"
side = "both"

[download]
url = "https://cdn.modrinth.com/data/nfrZ4xzm/versions/abc123/jei-1.20.1-forge-15.2.0.27.jar"
hash-format = "sha512"
hash = "0000..."

[option]
optional = true
default = true
description = "Just Enough Items"

[update.modrinth]
mod-id = "nfrZ4xzm"
version = "abc123"
```

字段：

| 字段 | 必填 | 说明 |
|---|---|---|
| `name` | 是 | 显示名 |
| `filename` | 是 | 相对元文件的目标路径 |
| `download` | 是 | `{ url, hash-format, hash }` |
| `side` | 否 | `both` / `client` / `server` |
| `option` | 否 | `{ optional, default, description }` |
| `update` | 否 | `update.curseforge{file-id,project-id}` / `update.modrinth{mod-id,version}` |

### 7.5 安装方式

```text
java -jar packwiz-installer-bootstrap.jar https://example.com/pack.toml
```

- 作为 Prism/MultiMC 的“启动前命令”使用，实现自动更新。
- 也支持独立运行。
- `packwiz-installer` 会读取 `pack.toml` → `index.toml` → 每个文件；`metafile = true` 的条目再读 `.pw.toml`。
- 文件按 `hash-format` 校验；已存在且哈希一致则跳过。
- `preserve = true` 的文件不会被覆盖。

### 7.6 坑

- `pack.toml` 里的 `index.hash` 必须随 `index.toml` 更新，否则安装器会认为索引被篡改。
- `filename` 是相对**元文件所在目录**的路径；不是相对包根。
- `murmur2` 是 CurseForge 指纹，不是通用内容哈希。
- packwiz 包本身不需要 ZIP；分发时直接给 `pack.toml` URL 即可。

---

## 8. HMCL 原生整合包

这是 HMCL 早期导出的格式，现代 HMCL 主要导出 MCBBS / Modrinth / 服务端包，但仍能导入。

### 8.1 结构

```text
pack.zip
├── modpack.json
└── minecraft/
    ├── pack.json
    ├── mods/
    ├── config/
    └── ...
```

### 8.2 `modpack.json`

```json
{
  "name": "Example Pack",
  "author": "ExampleAuthor",
  "version": "1.0.0",
  "gameVersion": "1.20.1",
  "description": "A small example pack."
}
```

字段对应 HMCL `Modpack` 类：`name`、`author`、`version`、`gameVersion`、`description`。

### 8.3 `minecraft/pack.json`

这是一个“游戏实例清单”，结构类似 Mojang 版本 JSON，常见字段：

```json
{
  "id": "1.20.1-forge-47.2.0",
  "inheritsFrom": "1.20.1",
  "mainClass": "cpw.mods.bootstraplauncher.BootstrapLauncher",
  "libraries": [],
  "arguments": { "game": [], "jvm": [] }
}
```

HMCL 读取 `pack.json` 的 `jar.id` 来确定游戏版本；如果 `jar` 为空，则回退到 `modpack.json.gameVersion`。

### 8.4 安装

- 把 `minecraft/` 的内容解压到实例根目录。
- 游戏本体和加载器仍需启动器下载/安装；这个格式只包含文件，不包含远端清单。
- 导入后 HMCL 在实例根写 `modpack.cfg`（`ModpackConfiguration` JSON），记录 `manifest/type/name/version/overrides[]`，用于更新检查。

### 8.5 与 MCBBS 的区别

- HMCL 原生：`modpack.json` + `minecraft/pack.json`。
- MCBBS：`mcbbs.packmeta` + `overrides/`，可远端下载。
- 两者都可能被叫“HMCL 整合包”，识别时看标记文件。

---

## 9. MCBBS 整合包（`mcbbs.packmeta`）

国内事实标准，起源于 HMCL 整合包格式，被 HMCL 3、FCL、PCL2、BakaXL 支持。

### 9.1 版本

| 版本 | 标记文件 | 说明 |
|---|---|---|
| v1 | `manifest.json` | 与 CurseForge 共用文件名，靠 `addons` 区分 |
| v2 | `mcbbs.packmeta` | 主清单；包内**还带一份 CurseForge 格式的 `manifest.json`**，方便 MultiMC 等导入 |

v2 的额外要求：整合包版本号必须满足 Maven/SemVer 规范。

### 9.2 结构

```text
pack.zip
├── mcbbs.packmeta          (v2)
├── manifest.json           (v2 附带，CurseForge 兼容；v1 时就是主清单)
└── overrides/
    ├── mods/
    ├── config/
    └── ...
```

### 9.3 字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `manifestType` | string | 是 | 固定 `"minecraftModpack"` |
| `manifestVersion` | int | 是 | v1 为 `1`，v2 为 `2` |
| `name` | string | 是 | 整合包名 |
| `version` | string | 是 | 版本（v2 要求 SemVer/Maven） |
| `author` | string | 是 | 作者 |
| `description` | string | 否 | 描述，可含 HTML |
| `fileApi` | string/null | 否 | 附加文件下载根地址 |
| `url` | string | 否 | 整合包介绍页 |
| `forceUpdate` | bool | 否 | 是否强制更新 |
| `origin` | array | 否 | 来源标注 `{ "type": "curseforge", "id": 123 }` |
| `addons` | array | 是 | 游戏/加载器组件 |
| `libraries` | array | 否 | 额外 Maven 库 |
| `files` | array | 是 | 附加文件列表 |
| `settings` | object | 否 | 安装开关 |
| `launchInfo` | object | 否 | 内存/Java/启动参数 |

`addons[]`：

```json
[
  { "id": "game", "version": "1.20.1" },
  { "id": "forge", "version": "47.2.0" }
]
```

常见 `id`：

| id | 含义 |
|---|---|
| `game` | Minecraft 版本（必需） |
| `forge` | Forge |
| `neoforge` | NeoForge |
| `fabric` | Fabric Loader |
| `quilt` | Quilt Loader |
| `liteloader` | LiteLoader |
| `optifine` | OptiFine |
| `cleanroom` | Cleanroom |
| `legacyfabric` | Legacy Fabric |

`files[]` 是带 `type` 的联合类型：

```json
[
  {
    "type": "addon",
    "path": "mods/example.jar",
    "hash": "0000000000000000000000000000000000000000",
    "force": true
  },
  {
    "type": "curse",
    "projectID": 238222,
    "fileID": 4593547,
    "fileName": "jei-1.20.1-forge-15.2.0.27.jar",
    "url": "",
    "force": false
  }
]
```

- `addon`：`path` 相对 `overrides/`；下载地址是 `fileApi + "/overrides/" + path`；`hash` 是 **SHA-1**。
- `curse`：按 CurseForge 的 `projectID/fileID` 下载；`fileName`、`url` 可选；`url` 为空时使用 CurseForge 页面地址兜底。
- `force`：是否强制覆盖/更新。

`settings`：

```json
{
  "install_mods": true,
  "install_resourcepack": true
}
```

`launchInfo`：

```json
{
  "minMemory": 2048,
  "supportJava": [17, 21],
  "launchArgument": ["--demo"],
  "javaArgument": ["-XX:+UseG1GC"]
}
```

- `minMemory`：最小内存（MB）。
- `supportJava`：支持的 Java 大版本列表。
- `launchArgument`：追加到游戏参数。
- `javaArgument`：追加到 JVM 参数。

### 9.4 `libraries`

`libraries` 使用 HMCL 的 Maven 库描述，常见字段：

```json
{
  "name": "com.example:example:1.0.0",
  "url": "https://example.com/maven/",
  "hint": "local",
  "filename": "example-1.0.0.jar"
}
```

`name` 是 Maven 坐标，`url` 是仓库根；`hint`/`filename` 是 MultiMC 风格的本地库提示。

### 9.5 安装流程

```text
1. 读 mcbbs.packmeta（没有则 manifest.json）
2. 从 addons 取 game 和加载器
3. 解压 overrides/ 到实例根
4. 对 files[]:
   - addon: 下载 fileApi + "/overrides/" + path，校验 SHA-1
   - curse: 走 CurseForge 解析
5. 应用 launchInfo 的 JVM/游戏参数
6. 应用 settings 的安装开关
```

### 9.6 坑

- v2 包里同时有 `mcbbs.packmeta` 和 `manifest.json`；**必须优先 `mcbbs.packmeta`**。
- v1 的 `manifest.json` 和 CurseForge 的 `manifest.json` 同名；看 `addons`。
- `fileApi` 可能为 `null`（全部文件内置）；此时不要拼 URL。
- `addon` 的 `hash` 是 SHA-1，不是 SHA-256。
- `path` 是相对 `overrides/` 的路径，不是相对包根。
- PCL2 不支持 MCBBS 里的 Quilt 加载器；HMCL 支持范围更广。
- `forceUpdate` 的语义各启动器略有差异；更新时以文件哈希为准更可靠。

---

## 10. HMCL 服务端整合包

用于服务端自动更新，HMCL/FCL 支持。

### 10.1 结构

```text
server-pack.zip
├── server-manifest.json
└── overrides/
    ├── mods/
    ├── config/
    └── ...
```

### 10.2 `server-manifest.json`

```json
{
  "name": "Example Server",
  "author": "ExampleAuthor",
  "version": "1.0.0",
  "description": "Server files.",
  "fileApi": "https://example.com/files",
  "files": [
    {
      "path": "mods/example.jar",
      "hash": "0000000000000000000000000000000000000000",
      "downloadURL": null
    }
  ],
  "addons": [
    { "id": "game", "version": "1.20.1" },
    { "id": "forge", "version": "47.2.0" }
  ]
}
```

字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `name` / `author` / `version` / `description` | string | 元信息 |
| `fileApi` | string | 文件根地址 |
| `files` | array | `{ path, hash, downloadURL }` |
| `addons` | array | 同 MCBBS，必须有 `game` |

- `hash` 是 SHA-1。
- HMCL 当前实现下载地址固定为 `fileApi + "/overrides/" + path`；`downloadURL` 是保留字段，不要假设所有实现都会使用它。
- ZIP 内 `overrides/` 解压到服务端根目录。
- 更新时按 `path` 对比本地 SHA-1。

---

## 11. ATLauncher `instance.json`

ATLauncher 的实例配置，**不是推荐的交换格式**；它导出时支持 CurseForge、Modrinth、两者都要、MultiMC。

### 11.1 结构

```text
instances/MyPack/
├── instance.json
├── mods/
├── config/
└── ...
```

`instance.json` 同时包含：

- ATLauncher 自己的 `launcher` 对象；
- 继承自 Minecraft 版本 JSON 的字段（`id`、`libraries`、`mainClass`、`inheritsFrom` 等）。

### 11.2 关键字段

```json
{
  "uuid": "00000000-0000-0000-0000-000000000000",
  "inheritsFrom": "1.20.1",
  "launcher": {
    "name": "Example Pack",
    "pack": "Example Pack",
    "description": "A small example pack.",
    "version": "1.0.0",
    "loaderVersion": {
      "version": "47.2.0",
      "rawVersion": "47.2.0",
      "recommended": false,
      "type": "Forge"
    },
    "maxMemory": 4096,
    "javaPath": null,
    "javaArguments": "",
    "mods": [],
    "overridePaths": ["config", "kubejs"],
    "checkForUpdates": true,
    "vanillaInstance": false
  }
}
```

`launcher` 常见字段：

| 字段 | 说明 |
|---|---|
| `name` / `pack` / `description` / `version` | 元信息 |
| `packId` / `externalPackId` | 平台项目 ID |
| `loaderVersion` | 加载器类型/版本 |
| `requiredMemory` / `maximumMemory` / `permGen` | 内存 |
| `javaPath` / `javaArguments` | Java 设置 |
| `mods` | `DisableableMod` 列表 |
| `overridePaths` | 导出时要带上的目录 |
| `curseForgeManifest` / `modrinthManifest` / `ftbPackManifest` / `technicModpack` | 来源清单缓存 |
| `checkForUpdates` | 是否检查更新 |
| `vanillaInstance` | 是否原版 |

### 11.3 导出

ATLauncher 的导出格式枚举：

```text
CURSEFORGE
MODRINTH
CURSEFORGE_AND_MODRINTH
MULTIMC
```

因此：

- 要交换给别的启动器，优先导出 CurseForge / Modrinth / MultiMC。
- `instance.json` 只适合 ATLauncher 自己，或做迁移时读取。

---

## 12. FTB 整合包

FTB 有两代完全不同的东西。

### 12.1 Legacy FTB

**目录/清单**：

- 官方目录是 XML（`modpacks.xml`），字段包括：
  `name`、`description`、`author`、`currentVersion`、`mcVersion`、`mods`、`logo`、`dir`、`file`、`oldVersions`、`packCode`、`type`。
- 整合包 ZIP 内是：

```text
pack.zip
└── minecraft/
    ├── pack.json          (版本 JSON，含 Forge 库)
    ├── mods/
    ├── config/
    ├── bin/modpack.jar    (旧版 Forge universal，可选)
    └── instMods/          (jar mod，可选)
```

Prism 的 Legacy FTB 安装器会：

1. 解压 ZIP；
2. 读 `minecraft/pack.json`；
3. 从 `libraries` 里找 `net.minecraftforge:forge` 或 `net.minecraftforge:fmlloader`，推出 Forge 版本；
4. 设置 `net.minecraft` 和 `net.minecraftforge` 组件。

### 12.2 FTB App（现代）

实例目录：

```text
~/.ftba/instances/MyPack/
├── instance.json
├── .ftbapp/
│   ├── version.json
│   └── logo
├── mods/
├── config/
└── ...
```

`instance.json` 关键字段（Prism 解析器要求）：

| 字段 | 类型 | 说明 |
|---|---|---|
| `uuid` | string | 实例 UUID |
| `id` | int | FTB 整合包 ID |
| `versionId` | int | FTB 版本 ID |
| `name` | string | 实例名 |
| `version` | string | 整合包版本 |
| `mcVersion` | string | Minecraft 版本 |
| `jvmArgs` | any | JVM 参数 |
| `totalPlayTime` | int | 总游玩时间 |
| `modLoader` | string | 如 `forge-47.2.0`；`loader-version` 形式 |

`.ftbapp/version.json`：

```json
{
  "targets": [
    { "name": "forge", "version": "47.2.0", "type": "modloader" },
    { "name": "minecraft", "version": "1.20.1", "type": "game" },
    { "name": "java", "version": "17.0.8", "type": "runtime" }
  ]
}
```

图标 `.ftbapp/logo` 是自定义二进制：**首字节表示图片类型**（`0x00` PNG、`0x01` JPG、`0x02` GIF、`0x03` WebP），后面是图片数据。

### 12.3 modpacks.ch API

```text
GET https://api.modpacks.ch/public/modpack/{packId}
GET https://api.modpacks.ch/public/modpack/{packId}/{versionId}
```

包对象常见字段：

```text
id, name, synopsis, description, type, featured, installs, plays,
updated, refreshed, art[], authors[], versions[], tags[], links, private
```

版本对象：

```json
{
  "id": 2,
  "name": "1.1.0",
  "type": "release",
  "parent": 1,
  "specs": { "id": 1, "minimum": 4096, "recommended": 6124 },
  "targets": [
    { "id": 1, "name": "forge", "type": "modloader", "version": "14.23.5.2846" },
    { "id": 2, "name": "minecraft", "type": "game", "version": "1.12.2" },
    { "id": -1, "name": "java", "type": "runtime", "version": "8.0.312+7" }
  ],
  "files": [
    {
      "id": 1486,
      "type": "resource",
      "path": "./animation",
      "name": "08.jpg",
      "version": "0",
      "url": "https://dist.modpacks.ch/...",
      "sha1": "...",
      "size": 18988,
      "clientOnly": null,
      "serverOnly": null,
      "optional": false,
      "curseforge": null
    }
  ]
}
```

- `files[].path` 可能是 `./animation` 这种形式，安装前要规范化。
- `files[].curseforge` 非空时是 `{ "project_id": ..., "file_id": ... }`，可走 CurseForge 下载。
- `clientOnly` / `serverOnly` / `optional` 控制侧别和可选。
- `targets` 同时给出 game / modloader / java。

### 12.4 FTB 与 CurseForge

现代 FTB 包也常发布到 CurseForge，此时下载到的就是 CurseForge `manifest.json` 格式。识别时不要假设“FTB = 某种专属格式”。

---

## 13. Technic / Solder

### 13.1 旧式 Technic 整合包 ZIP

```text
pack.zip
├── bin/
│   ├── modpack.jar          (Forge universal，旧)
│   └── version.json         (新版，可选)
├── mods/
├── config/
└── ...
```

Prism 的判定逻辑：

1. 如果有 `bin/modpack.jar`：
   - 打开这个 jar；
   - 如果里面有 `/version.json`，读出来；
   - 否则如果有 `forgeversion.properties`，用 `forge.major/minor/revision/build.number` 拼 Forge 版本；
   - 否则需要外部提供 Minecraft 版本，否则失败。
2. 否则如果有 `bin/version.json`，直接读。
3. 否则视为原版包，用外部提供的 Minecraft 版本。

`bin/modpack.jar` 保留为 jar mod 安装，同时把 Forge 作为组件记录。

### 13.2 Solder

Solder 是 Technic 的服务端分发型整合包。

只读 API：

```text
GET /api/modpack
GET /api/modpack/{slug}
GET /api/modpack/{slug}/{version}
```

`GET /api/modpack/{slug}` 返回：

```json
{
  "id": 1,
  "name": "hexxit",
  "display_name": "Hexxit",
  "url": "https://...",
  "icon": "https://...",
  "icon_md5": "...",
  "logo": "https://...",
  "logo_md5": "...",
  "background": "https://...",
  "background_md5": "...",
  "recommended": "1.0.10",
  "latest": "1.0.10",
  "builds": ["1.0.0", "1.0.10"]
}
```

`GET /api/modpack/{slug}/{version}` 返回：

```json
{
  "id": 1,
  "minecraft": "1.12.2",
  "java": "1.8",
  "java_runtime": null,
  "memory": 2048,
  "forge": "14.23.5.2847",
  "mods": [
    {
      "id": 1,
      "name": "rei-minimap",
      "version": "1.0.0",
      "md5": "0123456789abcdef0123456789abcdef",
      "filesize": 5242880,
      "url": "https://mods.example.com/mods/rei-minimap/rei-minimap-1.0.0.zip"
    }
  ]
}
```

- `include=mods` 会额外返回 `pretty_name`、`author`、`description`、`link`。
- 认证：`?cid=` 客户端 UUID，或 `?k=` API Key；私有包需要。
- 下载每个 mod 的 zip，校验 MD5，解压到实例根目录（zip 内部结构已经是 `mods/`、`config/` 等）。
- Solder 包没有统一的“清单文件”，元数据来自 API。
- Technic Platform 的公开 API 可能要求认证（实测 `/modpack/{slug}` 返回 401），实现时不要假设匿名可用。

---

## 14. GDLauncher `.gdlpack`

### 14.1 结构

```text
pack.gdlpack
├── gdlpack.json
├── .gdl/icon.png              (可选)
├── overrides/
├── clientOverrides/           (可选)
└── serverOverrides/           (可选)
```

### 14.2 `gdlpack.json`

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `formatVersion` | int | 是 | 当前 `1` |
| `name` | string | 是 | 包名 |
| `createdAt` | RFC3339 | 是 | 导出时间 |
| `dependencies` | object | 是 | `{ minecraft, modloaders[] }` |
| `entries` | array | 是 | 文件/可选功能 |
| `overrides` | string | 是 | 覆盖目录名，默认 `"overrides"` |
| `version` | string | 否 | 包版本 |
| `summary` | string | 否 | 简介 |
| `author` | string | 否 | 作者 |
| `icon` | string | 否 | 图标路径 |
| `serverOverrides` | string/null | 否 | 服务端覆盖目录 |
| `clientOverrides` | string/null | 否 | 客户端覆盖目录 |
| `source` | object | 否 | 来源标注 |

`dependencies`：

```json
{
  "minecraft": "1.20.1",
  "modloaders": [
    { "type": "forge", "version": "47.2.0", "primary": true }
  ]
}
```

`type` 取值：`forge`、`neoforge`、`fabric`、`quilt`。

`entries[]` 是联合类型：

```json
{
  "type": "platform",
  "hashes": {
    "sha512": "...",
    "sha1": "...",
    "murmur2": 123456789
  }
}
```

```json
{
  "type": "optional",
  "description": "可选光影",
  "platforms": [
    { "sha512": "...", "sha1": "...", "murmur2": 987654321 }
  ],
  "overridePaths": ["config/iris", "shaderpacks/default"]
}
```

- `platform`：通过哈希到 Modrinth（sha512）或 CurseForge（murmur2）反查并下载。
- `optional`：用户勾选后，下载 `platforms` 里的文件，并解压 `overridePaths` 指定的覆盖子路径。
- 三个哈希都要求存在。

### 14.3 两种导出

| 模式 | 行为 |
|---|---|
| 不打包（Bundle off） | 能反查到的文件写成 `platform`，jar 不放进包；反查不到的放进 `overrides/mods/` |
| 打包（Bundle on） | 不写 `platform`，所有 jar 都进 `overrides/mods/`，包更大但可离线 |

### 14.4 坑

- `source` 只是标注，不参与下载。
- 反查依赖平台可用性；源文件被删除会导致安装失败。
- `overridePaths` 是相对 `overrides/` 的路径，不是相对包根。

---

## 15. PCL2 的整合包处理

PCL2 没有独立的对外整合包格式；它是**导入器 + 导出器**。

### 15.1 导入识别

PCL2 的 `ModModpack.vb` 识别顺序（可直接作为实现参考）：

```text
从根目录：
1. mcbbs.packmeta          -> MCBBS（优先于 manifest.json）
2. mmc-pack.json           -> MultiMC（优先于 manifest.json）
3. modrinth.index.json     -> Modrinth
4. manifest.json:
     无 addons             -> CurseForge (PackType = 0)
     有 addons             -> MCBBS (PackType = 3)
5. modpack.json            -> HMCL (PackType = 1)
6. modpack.zip / modpack.mrpack -> 带启动器的压缩包 (PackType = 9)

从一级目录重复上述判断。

都没有 -> 假定为普通压缩包，走手动/兜底安装。
```

类型编号（PCL 内部）：`0 = CurseForge`、`1 = HMCL`、`2 = MultiMC`、`3 = MCBBS`、`4 = Modrinth`、`9 = 启动器包`。

### 15.2 安装细节

- **Modrinth**：只按 `sha1` 校验，不校验 `fileSize`（源码注释明确说明）；支持 `client-overrides`。
- **MCBBS**：解压 `overrides/`；从 `addons` 取 `game/forge/neoforge/fabric/optifine`；不支持 Quilt；`launchInfo` 写入 `PCL/Setup.ini`。
- **MultiMC**：优先 `minecraft/`，其次 `.minecraft/`；解析 `mmc-pack.json` 的 `net.minecraft`、`net.minecraftforge`、`net.neoforged`、`net.fabricmc.fabric-loader`；迁移 `instance.cfg` 的启动前命令、自动进服、图标、JVM 参数。
- **HMCL**：读 `modpack.json.gameVersion`，解压 `minecraft/`。
- **启动器包**：如果包内有 `modpack.zip` 或 `modpack.mrpack`，PCL 认为这是“带启动器的压缩包”，直接解压让用户选择文件夹，不按普通整合包安装。

### 15.3 导出

- 默认导出 ZIP，内部生成 **Modrinth 兼容的 `modrinth.index.json`**。
- 开启“Modrinth 上传模式”时导出 `.mrpack`。
- 可选择“打包资源文件”：不写下载 URL，直接把 jar 放进 `overrides/`。
- 可附带 PCL 启动器（“懒人包”），但这时就不是标准整合包。

---

## 16. 裸 ZIP / 懒人包 / 启动器包

### 16.1 裸 ZIP

没有清单，只有游戏目录：

```text
pack.zip
├── .minecraft/
│   ├── mods/
│   └── config/
└── ...
```

或：

```text
pack.zip
├── mods/
├── config/
└── ...
```

或：

```text
pack.zip
└── versions/
    └── MyPack/
        ├── MyPack.json
        └── mods/
```

安装器只能：

1. 找 `.minecraft/`、`minecraft/`、`versions/<id>/`、`mods/`；
2. 推断游戏版本（读版本 JSON / `versions/*/*.json`）；
3. 其余内容原样解压；
4. 让用户确认。

PCL2 对无法识别的包会走 `InstallPackCompress`，本质就是“解压 + 推断”。

### 16.2 懒人包 / 启动器包

常见形式：

- ZIP 内直接带一个完整的 `.minecraft` + 启动器 exe/jar；
- ZIP 内有 `modpack.zip` 或 `modpack.mrpack`；
- 自解压 EXE。

这类不是标准整合包，识别时要么按“裸 ZIP”，要么提示用户“这是启动器包，请解压后运行”。

### 16.3 RAR

PCL2 明确不支持 RAR；其他启动器也大多只支持 ZIP。遇到 `.rar` 应提示用户转成 ZIP。

---

## 17. 其他与旧格式

| 名称 | 情况 |
|---|---|
| Twitch / Curse App | 就是 CurseForge `manifest.json` |
| BakaXL | 没有自有交换格式；支持 CurseForge、Modrinth、MCBBS |
| GDLauncher 旧版 | 早期版本使用 CurseForge 清单；新版用 `.gdlpack` |
| MultiMC Legacy | 老实例 `InstanceType=Legacy`，不是 OneSix；现代工具一般不需要支持 |
| Technic Platform | 非 Solder 包通常是一个 ZIP；元数据在平台 API |
| MCUpdater | 小众；官方版用 XML 包定义，部分分支用 `pack.json`（`mcversion`、`mods[{modid,version,file,md5}]`）；不建议作为目标格式 |
| ATLauncher 自有包 | 通过 ATLauncher API 分发，不是通用交换格式 |
| 基岩版 `.mcaddon` / `.mcpack` / `.mcworld` | 不是 Java 整合包；不要混进 Java 启动器 |
| `.hmdp` | HMCL 某些版本的导出扩展名，本质 ZIP；按内容识别 |

---

## 18. 通用安装器设计

### 18.1 主流程

```text
install(archive):
    fmt, base = sniff(archive)

    manifest = parse(archive, base, fmt)

    game_version, loader = resolve_game_and_loader(manifest, fmt)
    ensure_vanilla(game_version)
    ensure_loader(game_version, loader)

    files = resolve_remote_files(manifest, fmt)
    files = filter_by_side_and_optional(files, user_choices)

    download_all(files, verify_hashes=True, safe_paths=True)

    extract_overrides_in_order(archive, base, fmt, instance_root)

    write_instance_metadata(fmt, manifest, instance_root)

    launch(instance_root)
```

### 18.2 解析层

建议为每种格式实现统一接口：

```text
interface ModpackAdapter {
    boolean matches(ZipFile zip, String base);
    ModpackManifest read(ZipFile zip, String base);
    List<RemoteFile> resolveFiles(ModpackManifest m);
    List<OverrideLayer> overrides(ModpackManifest m);
    InstanceMeta toInstanceMeta(ModpackManifest m);
}
```

统一中间模型：

```text
ModpackManifest {
    name, version, author, description
    mcVersion
    loaderType, loaderVersion
    List<RemoteFile> files
    List<OverrideLayer> overrides
    Set<String> optionalKeys
    sourceFormat
}

RemoteFile {
    path              // 相对实例根
    urls[]            // 按顺序尝试
    size
    hashes            // sha1/sha256/sha512/md5
    side              // client/server/both
    optional
    sourceId          // curseforge:project:file / modrinth:version / ...
}
```

### 18.3 加载器安装

不同加载器的“版本字符串”语义不同：

| 加载器 | 清单里的版本 | 安装时需要 |
|---|---|---|
| Forge | `47.2.0` | `1.20.1-47.2.0` 形式的 installer |
| NeoForge | `20.4.237` | `neoforge-20.4.237-installer.jar` |
| Fabric | `0.15.11` | loader + intermediary + profile JSON |
| Quilt | `0.23.1` | loader + hashed + profile JSON |
| LiteLoader | `1.12.2` | liteloader installer |
| OptiFine | `HD_U_...` | OptiFine jar + 启动参数 |
| Cleanroom | `0.2.4-alpha` | Cleanroom installer |

实现建议：

1. 统一先装原版（Mojang 版本 JSON + client jar + assets + libraries）。
2. 再装加载器（生成/合并版本 JSON）。
3. 最后合并 modpack 自己的 `libraries` / `patches` / 启动参数。

### 18.4 下载队列

- 并发 4–8 路（移动端建议 2–4）。
- 每个文件支持多 URL 回退。
- 超时、重试（指数退避）。
- 先写临时文件，校验通过再原子改名。
- 支持断点续传（Range）。
- 记录失败列表，最后汇总，不要因为一个 mod 失败就丢掉整个安装。

### 18.5 覆盖解压

```text
for layer in [overrides, side_overrides]:
    for entry in layer:
        target = safe_join(instance_root, entry.relative_path)
        ensure target starts with instance_root
        if entry is symlink: reject or materialize
        write file
```

顺序：基础覆盖先，侧别覆盖后；同名后写胜出。

### 18.6 实例元数据

安装完成后写一份自己的元数据，至少包含：

```json
{
  "format": "modrinth",
  "name": "...",
  "version": "...",
  "mcVersion": "1.20.1",
  "loader": { "type": "fabric", "version": "0.15.11" },
  "source": { "kind": "mrpack", "versionId": "..." },
  "optionalExcluded": ["mods/foo.jar"],
  "installedAt": "2026-10-29T12:00:00Z"
}
```

用于：

- 检查更新；
- 修复缺失文件；
- 卸载时知道哪些文件是整合包带来的。

---

## 19. 下载源与 API 速查

### 19.1 Mojang

```text
版本清单:
https://piston-meta.mojang.com/mc/game/version_manifest_v2.json

版本 JSON:
https://piston-meta.mojang.com/v1/packages/<sha1>/<version>.json

资源索引:
https://piston-meta.mojang.com/v1/packages/<sha1>/<assetIndex>.json

资源对象:
https://resources.download.minecraft.net/<前两位>/<hash>
```

### 19.2 CurseForge

```text
Base: https://api.curseforge.com
Header: x-api-key: <API_KEY>

GET  /v1/mods/{modId}
GET  /v1/mods/{modId}/files
GET  /v1/mods/{modId}/files/{fileId}
POST /v1/mods/files                 body: {"fileIds":[...]}
GET  /v1/mods/{modId}/files/{fileId}/download-url
```

CDN：

```text
https://edge.forgecdn.net/files/{fileId/1000}/{fileId%1000}/{fileName}
https://mediafilez.forgecdn.net/files/{fileId/1000}/{fileId%1000}/{fileName}
```

注意：`downloadUrl` 可能为 `null`；`fileStatus` 可能表示审核/删除；API Key 不要硬编码在客户端。

### 19.3 Modrinth

```text
Base: https://api.modrinth.com/v2

GET /project/{id|slug}
GET /project/{id|slug}/version
GET /version/{versionId}
GET /version_file/{hash}?algorithm=sha1
GET /version_file/{hash}?algorithm=sha512
```

建议 Header：

```text
User-Agent: MyLauncher/1.0.0 (contact@example.com)
```

### 19.4 FTB

```text
https://api.modpacks.ch/public/modpack/{packId}
https://api.modpacks.ch/public/modpack/{packId}/{versionId}
```

### 19.5 Solder

```text
{base}/api/modpack
{base}/api/modpack/{slug}
{base}/api/modpack/{slug}/{version}
{base}/api/modpack/{slug}/{version}?include=mods
```

认证参数：`?cid=` 或 `?k=`。

### 19.6 MCBBS / HMCL 服务端包

```text
{fileApi}/overrides/{path}
```

SHA-1 校验。`fileApi` 为 `null` 时说明文件全部内置，不要拼接。

### 19.7 加载器元数据

```text
Forge promotions:
https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json

Forge installer:
https://files.minecraftforge.net/maven/net/minecraftforge/forge/{mc}-{forge}/forge-{mc}-{forge}-installer.jar

NeoForge Maven:
https://maven.neoforged.net/releases/net/neoforged/neoforge/{version}/neoforge-{version}-installer.jar

Fabric meta:
https://meta.fabricmc.net/v2/versions/loader/{mc}/{loader}/profile/json

Quilt meta:
https://meta.quiltmc.org/v3/versions/loader/{mc}/{loader}/profile/json
```

---

## 20. 加载器版本规则

| 加载器 | 版本示例 | 说明 |
|---|---|---|
| Forge | `47.2.0` / `1.20.1-47.2.0` | 清单通常写前者；installer 用后者 |
| NeoForge | `20.4.237` / `21.1.77` | 前半段对应 MC 次版本；installer 同名 |
| Fabric | `0.15.11` | 还要 intermediary；版本 JSON 由 meta 生成 |
| Quilt | `0.23.1` | 还要 hashed；profile 由 meta 生成 |
| LiteLoader | `1.12.2` | 古老，通常与 Forge 组合 |
| OptiFine | `HD_U_I6` 等 | 不是标准 loader 组件；靠 jar + 参数 |
| Cleanroom | `0.2.4-alpha` | GTNH 生态 |

**规范化建议**：

```text
normalize_loader(name, version):
    name = lower(name)
    if name in ("neo-forge", "neoforge"):
        name = "neoforge"
    if name == "fabric":
        name = "fabric-loader"      # Modrinth 用后者
    if name == "quilt":
        name = "quilt-loader"
    if name in ("forge", "neoforge") and version matches r"^\d+\.\d+(\.\d+)?-.+$":
        # 可能是 "<mc>-<loader>"，去掉 mc 前缀
        version = version.split("-", 1)[1]
    return name, version
```

> 注意：Forge 的 `<mc>-<forge>` 里 MC 版本本身可能带 `-`（如 `1.20.1` 不带，但快照可能带）。更稳妥的做法是：如果版本字符串以 `<mcVersion>-` 开头，才去掉。

---

## 21. 安全清单

整合包是不可信输入，必须按“恶意 ZIP”处理。

### 21.1 路径

- 拒绝绝对路径：`/foo`、`\foo`、`C:\foo`、`C:/foo`。
- 拒绝 `..` 路径段（含 URL 编码、反斜杠变体）。
- 规范化后必须仍在实例根目录内。
- 拒绝 NUL 字节。
- 注意大小写不敏感文件系统上的同名冲突（Windows/部分 Android 存储）。
- 拒绝或安全处理 ZIP 符号链接条目（`unix mode` 为 symlink）。
- 拒绝设备文件、FIFO 等特殊条目。

### 21.2 解压

- 限制总解压大小和单文件大小（防 ZIP bomb）。
- 限制条目数量。
- 限制压缩比。
- 拒绝加密 ZIP（或明确提示）。
- 处理重复条目：后写覆盖前写，但要记录。
- 先解压到临时目录，校验后再移动到实例目录。

### 21.3 网络

- 只允许 HTTPS；HTTP 需显式白名单。
- 限制重定向次数（3–5 次），重定向后仍校验协议。
- 校验最终 URL 的域名（可选白名单）。
- 校验哈希；SHA-1/MD5 只用于兼容旧格式，能升级就升级。
- 不要执行下载来的任意脚本；加载器 installer 只从官方源下载。
- 注意 `file://`、`gopher://` 等协议注入。

### 21.4 内容

- `options.txt`、`servers.dat`、`config/` 都可能包含敏感信息或恶意配置。
- 不要自动执行 `PreLaunchCommand` / `PostExitCommand`，除非用户明确确认。
- Java 参数可能包含 `-javaagent`，加载任意 jar；展示给用户。
- 整合包可以包含资源包/光影，注意 ZIP 内的路径穿越。
- 不要把 API Key 写进日志。

### 21.5 更新

- 按哈希判断文件是否变化，不要只看版本号。
- 删除文件时只能删除自己记录过的文件，不要递归删实例目录。
- `preserve` / 用户修改要尊重。

---

## 22. 格式转换矩阵

| 从 \ 到 | CurseForge | Modrinth | MultiMC | MCBBS | packwiz | 裸 ZIP |
|---|---|---|---|---|---|---|
| **CurseForge** | — | 需哈希反查 Modrinth | 需下载全部文件 | 可直接映射 | 需哈希反查 | 下载全部文件 |
| **Modrinth** | 需哈希反查 CurseForge | — | 可直接映射 | 可直接映射 | 可直接生成 | 下载全部文件 |
| **MultiMC** | 需从 mods 反查 | 需从 mods 反查 | — | 需反查 | 需反查 | 直接打包 |
| **MCBBS** | 可直接映射 | 需反查 | 需下载 | — | 需反查 | 直接打包 |
| **packwiz** | 有导出工具 | 有导出工具 | 作为安装器 | 需转换 | — | 直接打包 |
| **裸 ZIP** | 需反查 | 需反查 | 可包装 | 可包装 | 需生成 metafile | — |

### 22.1 关键约束

- **CurseForge ID → Modrinth**：没有官方映射。只能用文件哈希（SHA-1/SHA-512）查 Modrinth，或用 Murmur2 反查 CurseForge 后再查。
- **Modrinth → CurseForge**：同样需要哈希（Murmur2 / SHA-1）反查。
- **MCBBS → CurseForge**：`curse` 类型可直接映射；`addon` 类型只能放进 `overrides/` 或上传到 CurseForge。
- **HMCL 原生 → 任何远端格式**：没有文件 ID/URL，只能全部放进 `overrides/`，或先算哈希反查。
- **任何格式 → 裸 ZIP**：永远可行，下载/内置全部文件即可。
- **离线包**：只有 MultiMC、HMCL 原生、打包模式的 GDLauncher、以及“带 overrides 的全部文件”能做到真正离线。

### 22.2 转换时的元数据

至少保留：

```text
name, version, author, description
mcVersion, loaderType, loaderVersion
每个文件的 path, size, hashes, side, optional
来源 ID（若有）
```

丢失来源 ID 会导致无法自动更新，只能整包替换。

---

## 23. 启动器实现对照

| 启动器 | 支持的导入格式 | 关键源码/行为 |
|---|---|---|
| **HMCL** | MCBBS、CurseForge、Modrinth、HMCL 原生、MultiMC、Server pack | `ModpackHelper.readModpackManifest` 顺序：MCBBS → Curse → Modrinth → HMCL → MultiMC → Server；Curse 加载器按 `forge-`/`fabric-`/`neoforge-` 前缀切；MultiMC 支持根目录或一层目录 |
| **FCL** | 同 HMCL（六种） | 直接复用 HMCL 的 provider 结构；Android 端 |
| **PCL2** | MCBBS、MultiMC、Modrinth、CurseForge、HMCL、启动器包、裸 ZIP | `ModModpack.vb`；识别顺序见 15.1；导出 Modrinth 兼容 ZIP/.mrpack |
| **Prism / MultiMC** | CurseForge、Modrinth、MultiMC、Technic、FTB、packwiz（通过 installer）、ATLauncher 导出的 MultiMC 包 | Technic 读 `bin/modpack.jar`/`bin/version.json`；FTB App 读 `instance.json`；Legacy FTB 读 `minecraft/pack.json` |
| **ATLauncher** | CurseForge、Modrinth、MultiMC、FTB、Technic、自有实例 | `Instance.java` / `InstanceLauncher.java`；导出枚举 `CURSEFORGE/MODRINTH/CURSEFORGE_AND_MODRINTH/MULTIMC` |
| **GDLauncher** | `.gdlpack`、CurseForge、Modrinth | `gdlpack.json`；哈希反查 |
| **BakaXL** | CurseForge、Modrinth、MCBBS | 无自有交换格式 |
| **Technic Launcher** | Technic ZIP、Solder | `bin/modpack.jar` / API |

### 23.1 各启动器的“私有落盘”

| 启动器 | 实例元数据 |
|---|---|
| HMCL | `versions/<id>/modpack.cfg`（`ModpackConfiguration` JSON） |
| FCL | 同 HMCL 系 |
| Prism | `instance.cfg` + `mmc-pack.json` |
| PCL2 | `PCL/Setup.ini` 等，不是交换格式 |
| ATLauncher | `instance.json` |
| FTB App | `instance.json` + `.ftbapp/version.json` |
| GDLauncher | 自身数据库/配置 |

---

## 24. 参考来源

### 官方/规范

- Modrinth, “Modrinth Modpack Format (.mrpack)”: <https://support.modrinth.com/en/articles/8802351-modrinth-modpack-format-mrpack>
- Modrinth API: <https://docs.modrinth.com/api/>
- packwiz 文档: <https://packwiz.infra.link/reference/pack-format/pack-toml/>、<https://packwiz.infra.link/reference/pack-format/index-toml/>、<https://packwiz.infra.link/reference/pack-format/mod-toml/>
- MultiMC Wiki, “Export Instance”: <https://github.com/MultiMC/wiki/blob/master/Export-Instance.md>
- MultiMC Wiki, “JSON Patches”: <https://github.com/MultiMC/wiki/blob/master/JSON-Patches.md>
- CurseForge API: <https://docs.curseforge.com/rest-api/>
- Solder API: <https://docs.solder.io/api/read/modpacks/>、<https://docs.solder.io/api/read/builds/>
- FTB API: <https://api.modpacks.ch/public/modpack/1>
- GDLauncher, “Modpack Manifest Format”: <https://gdlauncher.com/docs/modpack-manifest-format/>

### 源码

- HMCL: <https://github.com/HMCL-dev/HMCL>
  - `HMCLCore/.../modpack/ModpackProvider.java`
  - `HMCLCore/.../modpack/curse/CurseManifest.java`
  - `HMCLCore/.../modpack/modrinth/ModrinthManifest.java`
  - `HMCLCore/.../modpack/mcbbs/McbbsModpackManifest.java`
  - `HMCLCore/.../modpack/multimc/MultiMCManifest.java`
  - `HMCLCore/.../modpack/server/ServerModpackManifest.java`
  - `HMCL/src/main/java/org/jackhuang/hmcl/game/ModpackHelper.java`
- FCL: <https://github.com/FCL-Team/FoldCraftLauncher>
- PCL2: <https://github.com/Meloong-Git/PCL>（`Plain Craft Launcher 2/Modules/Minecraft/ModModpack.vb`）
- Prism Launcher: <https://github.com/PrismLauncher/PrismLauncher>
  - `launcher/modplatform/ftb/FTBPackManifest.h`
  - `launcher/modplatform/import_ftb/PackHelpers.cpp`
  - `launcher/modplatform/legacy_ftb/PackInstallTask.cpp`
  - `launcher/modplatform/technic/TechnicPackProcessor.cpp`
  - `launcher/modplatform/technic/SolderPackManifest.h`
- ATLauncher: <https://github.com/ATLauncher/ATLauncher>
  - `src/main/java/com/atlauncher/data/Instance.java`
  - `src/main/java/com/atlauncher/data/InstanceLauncher.java`
  - `src/main/java/com/atlauncher/data/InstanceExportFormat.java`
- GDLauncher 文档中的 `gdlpack.json` 说明
- MCBBS v2 说明: <https://github.com/HMCL-dev/HMCL/issues/964>

### 说明

- CurseForge 的 `manifest.json` 没有公开的正式 JSON Schema；本文依据 CurseForge App、HMCL、ATLauncher、GDLauncher、libium 等实现归纳。
- MCBBS 规范没有公开的单一标准文档；本文依据 HMCL/FCL/PCL2 的解析实现和 MCBBS v2 说明。
- 各启动器对“缺失字段”的默认值并不一致，凡是关键字段都应显式写入。

---

## 附录 A：跨格式字段对照

| 概念 | CurseForge | Modrinth | MultiMC | packwiz | MCBBS | Server pack | GDLauncher | FTB App |
|---|---|---|---|---|---|---|---|---|
| 名称 | `name` | `name` | `instance.cfg:name` | `name` | `name` | `name` | `name` | `name` |
| 版本 | `version` | `versionId` | — | `version` | `version` | `version` | `version` | `version` |
| 作者 | `author` | — | — | `author` | `author` | `author` | `author` | — |
| 简介 | `description` | `summary` | `notes` | `description` | `description` | `description` | `summary` | `synopsis` |
| MC 版本 | `minecraft.version` | `dependencies.minecraft` | `net.minecraft` 组件 | `versions.minecraft` | `addons[game]` | `addons[game]` | `dependencies.minecraft` | `mcVersion` |
| 加载器 | `minecraft.modLoaders[].id` | `dependencies.{forge,neoforge,fabric-loader,quilt-loader}` | `components` | `versions.*` | `addons` | `addons` | `dependencies.modloaders[]` | `modLoader` |
| 文件列表 | `files[]`（ID） | `files[]`（URL+hash） | 全部内置 | `index.toml` + metafile | `files[]` | `files[]` | `entries[]` | API `files[]` |
| 可选 | `required:false` | `env.optional` | 无 | `option.optional` | 无（或 `force`） | 无 | `entries[type=optional]` | `optional` |
| 侧别 | 无 | `env` | 无 | `side` | 无 | 无 | `clientOverrides/serverOverrides` | `clientOnly/serverOnly` |
| 哈希 | 通常无 | sha1 + sha512 | 无 | 多算法 | SHA-1（addon） | SHA-1 | sha512/sha1/murmur2 | sha1 |
| 覆盖 | `overrides` | `overrides/` 等 | `minecraft/` | 直接写根 | `overrides/` | `overrides/` | `overrides/` 等 | 目录内文件 |
| 更新源 | CF ID | Modrinth version | 平台托管键 | `update` | `fileApi` | `fileApi` | 哈希反查 | modpacks.ch |

---

## 附录 B：本目录附带的 schema 与示例

```text
docs/
├── Minecraft整合包格式全解.md
├── schemas/
│   ├── curseforge.manifest.schema.json
│   ├── modrinth.index.schema.json
│   ├── mcbbs.packmeta.schema.json
│   ├── mmc-pack.schema.json
│   ├── server-manifest.schema.json
│   └── gdlpack.schema.json
└── examples/
    ├── curseforge.manifest.json
    ├── modrinth.index.json
    ├── mcbbs.packmeta.json
    ├── mmc-pack.json
    ├── instance.cfg
    ├── server-manifest.json
    ├── gdlpack.json
    ├── pack.toml
    ├── index.toml
    └── example.pw.toml
```

这些示例用于：

- 单元测试的输入；
- 格式嗅探的回归样本；
- 生成器/转换器的目标结构。

> 示例里的哈希都是占位值（全 0 或示意值），不要用于真实校验。

---

## 附录 C：待确认/版本差异

以下是实现时需要留意的“不同版本可能不同”的点：

1. **CurseForge `required` 缺失**：HMCL 视为 `false`，其他实现可能视为 `true`。
2. **NeoForge 的 CurseForge ID**：`neoforge-20.4.237` 与 `neoforge-1.20.1-47.1.106` 两种写法都存在。
3. **Modrinth `fabric-loader` 与 `fabric`**：规范是 `fabric-loader`，部分第三方包会写 `fabric`，建议兼容。
4. **Modrinth `formatVersion`**：规范当前为 `1`；未来可能升级，遇到更高版本应提示而不是崩溃。
5. **MultiMC 与 Prism 的游戏目录**：`minecraft/` 与 `.minecraft/` 都可能；两者都有时 Prism 优先 `minecraft/`。
6. **MCBBS v1/v2**：v2 才有 `mcbbs.packmeta`；v1 的 `manifest.json` 与 CurseForge 同名。
7. **HMCL 原生格式**：现代 HMCL 已不作为导出目标，只保证导入。
8. **PCL2 导出**：默认 ZIP 内是 Modrinth 清单，不是 CurseForge；不要按扩展名判断。
9. **FTB Legacy 与 FTB App**：完全是两套东西；`instance.json` 是 FTB App 的，`minecraft/pack.json` 是 Legacy 的。
10. **Technic Platform API**：公开读取可能需要认证；Solder 才是稳定的只读 API。
11. **packwiz**：`pack-format` 可能升级；`index.hash` 必须随索引更新。
12. **GDLauncher**：`.gdlpack` 是较新的格式；旧版可能仍是 CurseForge 清单。
13. **ATLauncher `instance.json`**：字段会随版本增加；只把它当内部格式，不要当标准。
14. **HMCL 服务端包**：HMCL 当前固定按 `fileApi + "/overrides/" + path` 下载；`downloadURL` 是保留字段，不要假设它一定被使用。
15. **压缩包编码**：ZIP 内文件名可能是 GBK/UTF-8；中文包要处理编码（HMCL/FCL 有 charset 参数）。
