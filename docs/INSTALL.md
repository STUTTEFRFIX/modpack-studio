# 安装与构建

仓库不提交 APK、EXE 和 `.mrpack`。请从源码构建，或使用维护者提供的 Release 附件。

## Android APK

前置：

- Android SDK 的 `android.jar`
- `aapt2`、`d8`、`apksigner`、`javac`、`keytool`、`zip`

```sh
ANDROID_JAR=/path/to/android.jar ./scripts/build_apk.sh
```

输出：

```text
dist/ModpackStudio-dev.apk
```

安装：

```sh
adb install -r dist/ModpackStudio-dev.apk
```

或把 APK 传到手机，在文件管理器里安装（允许未知来源）。

- 应用名：整合包工作台
- 包名：`com.modpackstudio.app`
- Android 7.0+（minSdk 24 / targetSdk 34）
- 默认使用脚本生成的调试签名；正式发布请替换 release keystore
- 导入 `.mrpack` 需要访问 `https://api.modrinth.com`
- 导入 CurseForge `.zip` 需要访问 `https://api.curseforge.com`，并在首次导入时填入自己的 API Key（存在本机）

## Windows EXE

```sh
./scripts/build_desktop.sh
```

输出：

```text
dist/ModpackStudio.exe
dist/modpack-studio-desktop-linux
```

Windows 版是控制台程序：

1. 双击启动本地服务
2. 自动打开默认浏览器
3. 关闭控制台即停止

可选参数：

```text
ModpackStudio.exe --no-browser
ModpackStudio.exe --port 18080
```

未签名 EXE 可能触发 SmartScreen；正式发布建议签名。

## Web

直接打开：

```text
internal/webui/graph.html
```

或者：

```sh
go run ./cmd/modpack-studio-desktop
```

## 运行时限制

- 导入支持 Modrinth `.mrpack`、CurseForge `.zip`（清单型需要 API Key）、带 `mods/*.jar` 的启动器导出
- MCBBS / packwiz 的清单型包还不能在运行时导入（没有 jar 可读）
- 中文翻译走 MyMemory 公共接口，匿名额度约 5000 字符/天；默认只译简介 + 正文前 700 字，结果缓存在本机
- 翻译需要访问 `https://api.mymemory.translated.net`
- 暂不支持编辑、更新、卸载、导出
- 中文译名只覆盖 `experiments/graph-demo/zh_names.json` 中已有的 slug

## 发布检查清单

见 [HANDOVER.md](HANDOVER.md) 的“构建与发版”一节。
