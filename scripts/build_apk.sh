#!/usr/bin/env bash
# Build the WebView APK. Requires Android build-tools and an android.jar.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ANDROID_JAR="${ANDROID_JAR:-$ROOT/tools/android.jar}"
ANDROID_DIR="$ROOT/android"
DIST="$ROOT/dist"
KEYSTORE="${KEYSTORE:-$ANDROID_DIR/debug.keystore}"
KEY_ALIAS="${KEY_ALIAS:-modpackstudio}"
KS_PASS="${KS_PASS:-modpackstudio}"
KEY_PASS="${KEY_PASS:-modpackstudio}"

if [ ! -f "$ANDROID_JAR" ]; then
  echo "android.jar not found: $ANDROID_JAR" >&2
  echo "Set ANDROID_JAR=/path/to/android.jar (Android SDK platform)." >&2
  exit 1
fi

for tool in javac d8 aapt2 apksigner zip keytool; do
  command -v "$tool" >/dev/null 2>&1 || { echo "missing tool: $tool" >&2; exit 1; }
done

if [ ! -f "$ROOT/internal/webui/graph.html" ]; then
  echo "missing internal/webui/graph.html; run scripts/build_demo.sh first" >&2
  exit 1
fi

mkdir -p "$DIST"
cp "$ROOT/internal/webui/graph.html" "$ANDROID_DIR/assets/graph.html"
cd "$ANDROID_DIR"

rm -rf build
mkdir -p build/classes build/dex

javac -source 8 -target 8 -classpath "$ANDROID_JAR" \
  -d build/classes src/com/modpackstudio/app/MainActivity.java

# Include EVERY generated class, including MainActivity$1.class etc.
d8 --lib "$ANDROID_JAR" --min-api 24 --output build/dex \
  $(find build/classes -name '*.class')

aapt2 compile --dir res -o build/res.zip
aapt2 link -o build/base.apk -I "$ANDROID_JAR" \
  --manifest AndroidManifest.xml -A assets build/res.zip

(cd build && zip -j base.apk dex/classes.dex >/dev/null)

if [ ! -f "$KEYSTORE" ]; then
  echo "generating debug keystore: $KEYSTORE"
  keytool -genkeypair -keystore "$KEYSTORE" -alias "$KEY_ALIAS" \
    -keyalg RSA -keysize 2048 -validity 10000 \
    -storepass "$KS_PASS" -keypass "$KEY_PASS" \
    -dname "CN=Modpack Studio, OU=Dev, O=Modpack Studio, C=CN" >/dev/null 2>&1
fi

OUT="$DIST/ModpackStudio-dev.apk"
apksigner sign --ks "$KEYSTORE" --ks-key-alias "$KEY_ALIAS" \
  --ks-pass "pass:$KS_PASS" --key-pass "pass:$KEY_PASS" \
  --out "$OUT" build/base.apk

echo "APK: $OUT"
echo "sha256: $(sha256sum "$OUT" | cut -d' ' -f1)"
