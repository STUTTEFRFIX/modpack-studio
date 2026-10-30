package pack

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

func zipReader(t *testing.T, entries map[string]string) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return zr
}

func TestSniff(t *testing.T) {
	const cf = `{"manifestType":"minecraftModpack","manifestVersion":1,"name":"x","version":"1","author":"a","minecraft":{"version":"1.20.1","modLoaders":[{"id":"forge-47.2.0","primary":true}]},"files":[]}`
	tests := []struct {
		name    string
		entries map[string]string
		want    model.Format
		base    string
	}{
		{"curseforge", map[string]string{"manifest.json": cf}, model.FormatCurseForge, ""},
		{"mcbbs v2", map[string]string{"mcbbs.packmeta": `{"manifestType":"minecraftModpack","manifestVersion":2,"addons":[],"files":[]}`}, model.FormatMCBBS, ""},
		{"mcbbs v1", map[string]string{"manifest.json": `{"manifestType":"minecraftModpack","manifestVersion":1,"addons":[],"files":[]}`}, model.FormatMCBBS, ""},
		{"modrinth", map[string]string{"modrinth.index.json": `{"formatVersion":1,"game":"minecraft","versionId":"1","name":"x","files":[],"dependencies":{"minecraft":"1.20.1"}}`}, model.FormatModrinth, ""},
		{"multimc", map[string]string{"instance.cfg": "[General]\nname=X\n", "mmc-pack.json": `{"formatVersion":1,"components":[]}`}, model.FormatMultiMC, ""},
		{"hmcl", map[string]string{"modpack.json": `{}`, "minecraft/pack.json": `{}`}, model.FormatHMCL, ""},
		{"server", map[string]string{"server-manifest.json": `{}`}, model.FormatServer, ""},
		{"gdlpack", map[string]string{"gdlpack.json": `{}`}, model.FormatGDLPack, ""},
		{"packwiz", map[string]string{"pack.toml": `name="x"`}, model.FormatPackwiz, ""},
		{"plain mods", map[string]string{"mods/foo.jar": ""}, model.FormatPlain, ""},
		{"plain dot minecraft", map[string]string{".minecraft/mods/foo.jar": ""}, model.FormatPlain, ".minecraft/"},
		{"nested curseforge", map[string]string{"MyPack/manifest.json": cf}, model.FormatCurseForge, "MyPack/"},
		{"manifest addons wins", map[string]string{"manifest.json": `{"manifestType":"minecraftModpack","addons":[]}`}, model.FormatMCBBS, ""},
		{"ftb app", map[string]string{"instance.json": `{"uuid":"u","versionId":1,"mcVersion":"1.20.1"}`}, model.FormatFTBApp, ""},
		{"atlauncher", map[string]string{"instance.json": `{"launcher":{"name":"x"}}`}, model.FormatATLauncher, ""},
		{"technic", map[string]string{"bin/version.json": `{}`}, model.FormatTechnic, ""},
		{"launcher bundle", map[string]string{"modpack.mrpack": ""}, model.FormatLauncherBundle, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sniff(zipReader(t, tt.entries))
			if got.Format != tt.want || got.Base != tt.base {
				t.Fatalf("Sniff() = %s base=%q, want %s base=%q", got.Format, got.Base, tt.want, tt.base)
			}
		})
	}
}
