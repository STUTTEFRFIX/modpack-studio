# Modpack Studio 0.2.1-import

> 0.2.0 的 APK 因构建时漏打包 `MainActivity$1` 等内部类，启动会崩溃；0.2.1 已修复并实机验证可启动。

## Android APK

```text
/sdcard/cccodroid/ModpackStudio-0.2.1-import.apk
```

- 应用名：整合包工作台
- 包名：`com.modpackstudio.app`
- versionCode 3 / versionName 0.2.1-import
- Android 7.0+（minSdk 24 / targetSdk 34）
- 已安装到本机（覆盖 0.1.0）
- 测试签名，不是应用商店发布包
- 卸载路径：设置 → 应用 → 整合包工作台

### 导入整合包

1. 点顶部「导入」
2. 选择 `.mrpack` 文件
3. 应用会解析 ZIP、查询 Modrinth、生成关系图并自动重载
4. 想回到内置示例，点左侧「返回示例」

当前支持：

- Modrinth `.mrpack`（含嵌套一层目录的 `modrinth.index.json`）
- 从 Modrinth API 拉取 mod 名称、分类、介绍、依赖关系
- 中文译名（内置 `zh_names.json` 映射；未知 mod 显示英文）
- 导入数据保存到应用私有文件，重启仍保留
- 选择文件使用系统文件选择器，不需要存储权限

暂不支持：

- CurseForge `manifest.json`、MCBBS、MultiMC、packwiz 的导入
- 编辑、更新、卸载、导出
- 中文译名的在线查询（未知 mod 需要手工补 `zh_names.json`）

## Windows EXE

```text
/sdcard/cccodroid/ModpackStudio-0.2.1-import.exe
```

- Windows 10/11 x64
- 双击运行，启动本地服务并打开默认浏览器
- 控制台窗口保持打开；关闭控制台即停止
- 可选参数：

```text
ModpackStudio-0.2.1-import.exe --no-browser
ModpackStudio-0.2.1-import.exe --port 18080
```

- 导入方式相同：点「导入」，选择 `.mrpack`
- 浏览器需要能访问 `https://api.modrinth.com`
- 已做 PE32+ 格式检查；同一服务代码在本机 Linux 构建上验证了页面和导入流程，Windows 实机尚未测试
- 未签名 EXE 可能触发 SmartScreen

## 验证情况

- 0.2.1 APK 已安装并实机启动验证；导入流程在 Chromium 无头浏览器中用 Create Live 5 实测通过
- 导入结果与 Python 原型一致：95 个节点、71 条边、51 条依赖、20 条联动、4 个枢纽、27 个孤立
- Android WebView 文件选择器和本地文件存储已接入；未在实机上完整走完一次「选文件 → 导入」流程
- Windows EXE 未在 Windows 实机运行

## 校验值

```text
APK  sha256 982a0997c2b181b69608ba9f883903346fdb2c7dbf2dca90090387a950c5d22f
EXE  sha256 bda358f6c71c0b03fc90c21f7172df1cab24e74b91c960d29258aae97e6ec771
```

## 源码位置

```text
/data/user/0/cc.codroid/files/home/workspace/个人修行项目/modpack-studio/
```

- Android 工程：`android/`
- Windows/桌面入口：`cmd/modpack-studio-desktop/`
- 图数据与前端：`experiments/graph-demo/`
- Go 核心：`internal/`
