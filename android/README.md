# Android APK

一个 WebView 壳，加载 `assets/graph.html`，并接入：

- 系统文件选择器（导入 `.mrpack`）
- `AndroidHost` JS 接口（把导入数据保存到应用私有文件）
- 外部链接用系统浏览器打开

## 构建

需要：

- `android.jar`（Android SDK platform，仓库不包含）
- `aapt2`、`d8`、`apksigner`、`javac`、`keytool`、`zip`

```sh
ANDROID_JAR=/path/to/android.jar ../scripts/build_apk.sh
```

脚本会把 `../internal/webui/graph.html` 复制到 `assets/graph.html`，编译全部 Java class，签名并输出 APK。

## 关键坑

`javac` 会生成 `MainActivity$1.class` 等匿名内部类。`d8` 必须收集全部 class：

```sh
d8 ... $(find build/classes -name '*.class')
```

只传 `MainActivity.class` 会导致启动崩溃：

```text
NoClassDefFoundError: Lcom/modpackstudio/app/MainActivity$1;
```

## 文件

- `AndroidManifest.xml`：包名 `com.modpackstudio.app`，minSdk 24，targetSdk 34
- `src/com/modpackstudio/app/MainActivity.java`：WebView、文件选择器、AndroidHost 存储
- `res/drawable/icon.png`：图标
- `assets/graph.html`：构建时生成，仓库忽略

## 签名

默认使用脚本生成的调试 keystore。正式发布前替换为自己的 release keystore，并修改 `versionCode` / `versionName`。
