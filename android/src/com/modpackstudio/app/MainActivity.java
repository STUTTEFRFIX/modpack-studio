package com.modpackstudio.app;

import android.app.Activity;
import android.content.Intent;
import android.graphics.Color;
import android.net.Uri;
import android.os.Bundle;
import android.view.View;
import android.webkit.JavascriptInterface;
import android.webkit.ValueCallback;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;

import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;

public class MainActivity extends Activity {
    private WebView webView;
    private ValueCallback<Uri[]> filePathCallback;

    private static final int REQ_PICK = 1001;
    private static final int PAPER = Color.rgb(247, 244, 234);

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        getWindow().setStatusBarColor(PAPER);
        getWindow().setNavigationBarColor(PAPER);
        getWindow().getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR);

        webView = new WebView(this);
        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setAllowFileAccess(true);
        settings.setAllowContentAccess(true);
        settings.setCacheMode(WebSettings.LOAD_DEFAULT);
        settings.setUseWideViewPort(true);
        settings.setLoadWithOverviewMode(true);
        settings.setTextZoom(100);
        webView.setBackgroundColor(PAPER);
        webView.setWebViewClient(new WebViewClient() {
            @Override
            public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) {
                // 页面用 http://127.0.0.1/ 这个假地址加载，实际内容从 assets 直接喂，
                // 不监听任何端口。localhost 属于安全上下文，Worker 和 crypto.subtle 才可用。
                if (isLocalHost(request.getUrl())) {
                    try {
                        return new WebResourceResponse("text/html", "utf-8",
                                getAssets().open("graph.html"));
                    } catch (Exception e) {
                        return null;
                    }
                }
                return null;
            }

            @Override
            public boolean shouldOverrideUrlLoading(WebView view, String url) {
                if (url == null) return false;
                if (url.startsWith("http://127.0.0.1") || url.startsWith("https://127.0.0.1")) return false;
                if (url.startsWith("http://") || url.startsWith("https://")) {
                    try {
                        startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(url)));
                        return true;
                    } catch (Exception ignored) {
                        return false;
                    }
                }
                return false;
            }
        });
        webView.addJavascriptInterface(new Object() {
            @JavascriptInterface
            public String loadImported() {
                return readImportedFile();
            }

            @JavascriptInterface
            public void saveImported(String json) {
                writeImportedFile(json);
            }

            @JavascriptInterface
            public void clearImported() {
                importedFile().delete();
            }
        }, "AndroidHost");
        webView.setWebChromeClient(new WebChromeClient() {
            @Override
            public boolean onShowFileChooser(WebView view,
                                             ValueCallback<Uri[]> callback,
                                             FileChooserParams params) {
                if (filePathCallback != null) {
                    filePathCallback.onReceiveValue(null);
                }
                filePathCallback = callback;
                if (!openDocumentPicker()) {
                    filePathCallback = null;
                    return false;
                }
                return true;
            }
        });
        setContentView(webView);
        webView.loadUrl(PAGE_URL);
    }

    private static final String PAGE_URL = "http://127.0.0.1/graph.html";

    private boolean isLocalHost(Uri uri) {
        return uri != null && "127.0.0.1".equals(uri.getHost());
    }

    /**
     * 优先用 SAF 文档选择器（ACTION_OPEN_DOCUMENT）。ColorOS 会把
     * ACTION_GET_CONTENT 路由到它自己的媒体选择器，只给“照片和视频/更多”，
     * 根本翻不到 Downloads 里的 .zip。找不到文档选择器时再退回旧方式。
     */
    private boolean openDocumentPicker() {
        Intent doc = new Intent(Intent.ACTION_OPEN_DOCUMENT);
        doc.addCategory(Intent.CATEGORY_OPENABLE);
        doc.setType("*/*");
        doc.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
        try {
            startActivityForResult(doc, REQ_PICK);
            return true;
        } catch (Exception first) {
            // fall through
        }
        Intent legacy = new Intent(Intent.ACTION_GET_CONTENT);
        legacy.addCategory(Intent.CATEGORY_OPENABLE);
        legacy.setType("*/*");
        legacy.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
        try {
            startActivityForResult(legacy, REQ_PICK);
            return true;
        } catch (Exception second) {
            return false;
        }
    }

    /**
     * 把选中的 content:// 文件先拷到应用缓存，再把 file:// URI 交给 WebView。
     * 有些系统 / 网盘返回的 URI 是占位符或者关掉流就失效，直接交给 WebView 会读到 0 字节。
     */
    private Uri copyPickedToCache(Uri src) {
        if (src == null) return null;
        if ("file".equals(src.getScheme())) return src;
        File out = new File(getCacheDir(), "picked-" + System.currentTimeMillis() + ".zip");
        try (InputStream in = getContentResolver().openInputStream(src);
             FileOutputStream os = new FileOutputStream(out)) {
            if (in == null) return null;
            byte[] buf = new byte[65536];
            int n;
            while ((n = in.read(buf)) != -1) os.write(buf, 0, n);
            return Uri.fromFile(out);
        } catch (Exception e) {
            out.delete();
            return null;
        }
    }

    private File importedFile() {
        return new File(getFilesDir(), "imported.json");
    }

    private String readImportedFile() {
        File file = importedFile();
        if (!file.isFile()) return "";
        try (FileInputStream in = new FileInputStream(file);
             ByteArrayOutputStream out = new ByteArrayOutputStream()) {
            byte[] buffer = new byte[8192];
            int n;
            while ((n = in.read(buffer)) != -1) {
                out.write(buffer, 0, n);
            }
            return out.toString("UTF-8");
        } catch (Exception e) {
            return "";
        }
    }

    private void writeImportedFile(String json) {
        try (FileOutputStream out = new FileOutputStream(importedFile())) {
            out.write(json.getBytes(StandardCharsets.UTF_8));
        } catch (Exception ignored) {
        }
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        if (requestCode == REQ_PICK) {
            Uri[] result = null;
            if (resultCode == RESULT_OK && data != null) {
                if (data.getData() != null) {
                    Uri local = copyPickedToCache(data.getData());
                    result = new Uri[]{local != null ? local : data.getData()};
                } else if (data.getClipData() != null) {
                    int count = data.getClipData().getItemCount();
                    result = new Uri[count];
                    for (int i = 0; i < count; i++) {
                        Uri one = data.getClipData().getItemAt(i).getUri();
                        Uri local = copyPickedToCache(one);
                        result[i] = local != null ? local : one;
                    }
                }
            }
            if (filePathCallback != null) {
                filePathCallback.onReceiveValue(result);
                filePathCallback = null;
            }
            return;
        }
        super.onActivityResult(requestCode, resultCode, data);
    }

    @Override
    public void onBackPressed() {
        if (webView != null && webView.canGoBack()) {
            webView.goBack();
        } else {
            super.onBackPressed();
        }
    }
}
