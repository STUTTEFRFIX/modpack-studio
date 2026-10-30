package pack

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

func writeZipFile(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pack.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
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
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAnalyzeCurseForge(t *testing.T) {
	manifest := `{
		"manifestType":"minecraftModpack","manifestVersion":1,
		"name":"Example","version":"1.2.0","author":"Author",
		"minecraft":{"version":"1.20.1","modLoaders":[{"id":"forge-47.2.0","primary":true}]},
		"files":[
			{"projectID":1,"fileID":2,"required":true,"fileName":"a.jar"},
			{"projectID":3,"fileID":4,"required":false}
		],
		"overrides":"custom"
	}`
	p, err := Analyze(writeZipFile(t, map[string]string{
		"manifest.json":         manifest,
		"custom/mods/local.jar": "",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Format != model.FormatCurseForge || p.MCVersion != "1.20.1" {
		t.Fatalf("format/mc = %s/%s", p.Format, p.MCVersion)
	}
	if p.Loader.Type != "forge" || p.Loader.Version != "47.2.0" {
		t.Fatalf("loader = %+v", p.Loader)
	}
	if p.Stats.RemoteFiles != 2 || p.Stats.LocalJars != 1 || p.Stats.Optional != 1 {
		t.Fatalf("stats = %+v", p.Stats)
	}
	if p.Mods[0].Path != "mods/a.jar" || p.Mods[0].Source.ProjectID != "1" {
		t.Fatalf("remote mod = %+v", p.Mods[0])
	}
}

func TestAnalyzeModrinth(t *testing.T) {
	index := `{
		"formatVersion":1,"game":"minecraft","versionId":"1.0.0","name":"Example",
		"files":[
			{"path":"mods/api.jar","hashes":{"sha1":"1111111111111111111111111111111111111111","sha512":"2222"},
			 "env":{"client":"optional","server":"unsupported"},
			 "downloads":["https://cdn.modrinth.com/data/ABC/versions/VER/api.jar"],"fileSize":10}
		],
		"dependencies":{"minecraft":"1.20.1","fabric-loader":"0.15.11"}
	}`
	p, err := Analyze(writeZipFile(t, map[string]string{
		"modrinth.index.json":      index,
		"overrides/mods/local.jar": "",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Loader.Type != "fabric" || p.Loader.Version != "0.15.11" {
		t.Fatalf("loader = %+v", p.Loader)
	}
	if p.Stats.RemoteFiles != 1 || p.Stats.LocalJars != 1 || p.Stats.Optional != 1 {
		t.Fatalf("stats = %+v", p.Stats)
	}
	m := p.Mods[0]
	if m.Side != "client" || !m.Optional || m.Source.ProjectID != "ABC" || m.Source.VersionID != "VER" {
		t.Fatalf("mod = %+v", m)
	}
}

func TestAnalyzeMCBBS(t *testing.T) {
	meta := `{
		"manifestType":"minecraftModpack","manifestVersion":2,
		"name":"Example","version":"1.0.0","author":"Author",
		"fileApi":"https://example.com/files/",
		"addons":[{"id":"game","version":"1.20.1"},{"id":"forge","version":"47.2.0"}],
		"files":[
			{"type":"addon","path":"mods/a.jar","hash":"1111111111111111111111111111111111111111"},
			{"type":"curse","projectID":3,"fileID":4,"fileName":"b.jar"}
		]
	}`
	p, err := Analyze(writeZipFile(t, map[string]string{
		"mcbbs.packmeta":           meta,
		"manifest.json":            `{"manifestType":"minecraftModpack","manifestVersion":1,"files":[]}`,
		"overrides/mods/local.jar": "",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Loader.Type != "forge" || p.MCVersion != "1.20.1" {
		t.Fatalf("loader/mc = %+v/%s", p.Loader, p.MCVersion)
	}
	if p.Stats.RemoteFiles != 2 || p.Stats.LocalJars != 1 {
		t.Fatalf("stats = %+v", p.Stats)
	}
	if p.Mods[0].Source.URL != "https://example.com/files/overrides/mods/a.jar" {
		t.Fatalf("addon url = %q", p.Mods[0].Source.URL)
	}
}

func TestAnalyzeMultiMC(t *testing.T) {
	p, err := Analyze(writeZipFile(t, map[string]string{
		"instance.cfg": "[General]\nname=Example\nInstanceType=OneSix\n",
		"mmc-pack.json": `{"formatVersion":1,"components":[
			{"uid":"net.minecraft","version":"1.20.1"},
			{"uid":"net.minecraftforge","version":"47.2.0"}
		]}`,
		"minecraft/mods/local.jar": "",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Format != model.FormatMultiMC || p.Name != "Example" {
		t.Fatalf("pack = %+v", p)
	}
	if p.Loader.Type != "forge" || p.Loader.Version != "47.2.0" || p.Stats.LocalJars != 1 {
		t.Fatalf("loader/stats = %+v %+v", p.Loader, p.Stats)
	}
}

func TestAnalyzeHMCL(t *testing.T) {
	p, err := Analyze(writeZipFile(t, map[string]string{
		"modpack.json":             `{"name":"Example","version":"1.0","gameVersion":"1.20.1"}`,
		"minecraft/pack.json":      `{"id":"1.20.1"}`,
		"minecraft/mods/local.jar": "",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Format != model.FormatHMCL || p.MCVersion != "1.20.1" || p.Stats.LocalJars != 1 {
		t.Fatalf("pack = %+v", p)
	}
}

func TestAnalyzeServer(t *testing.T) {
	p, err := Analyze(writeZipFile(t, map[string]string{
		"server-manifest.json": `{
			"name":"Server","fileApi":"https://example.com/files",
			"files":[{"path":"mods/a.jar","hash":"1111111111111111111111111111111111111111"}],
			"addons":[{"id":"game","version":"1.20.1"},{"id":"forge","version":"47.2.0"}]
		}`,
		"overrides/mods/local.jar": "",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Loader.Type != "forge" || p.Stats.RemoteFiles != 1 || p.Stats.LocalJars != 1 {
		t.Fatalf("pack = %+v", p)
	}
	if p.Mods[0].Source.URL != "https://example.com/files/overrides/mods/a.jar" {
		t.Fatalf("url = %q", p.Mods[0].Source.URL)
	}
}

func TestAnalyzeGDLPack(t *testing.T) {
	p, err := Analyze(writeZipFile(t, map[string]string{
		"gdlpack.json": `{
			"formatVersion":1,"name":"Example","createdAt":"2026-10-30T00:00:00Z",
			"dependencies":{"minecraft":"1.20.1","modloaders":[{"type":"forge","version":"47.2.0","primary":true}]},
			"entries":[
				{"type":"platform","hashes":{"sha512":"aa","sha1":"bb","murmur2":1}},
				{"type":"optional","description":"shaders","platforms":[],"overridePaths":["shaderpacks"]}
			],
			"overrides":"overrides"
		}`,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Format != model.FormatGDLPack || p.Loader.Type != "forge" {
		t.Fatalf("pack = %+v", p)
	}
	if p.Stats.Entries != 2 || p.Stats.Optional != 1 {
		t.Fatalf("stats = %+v", p.Stats)
	}
}

func TestAnalyzePlain(t *testing.T) {
	p, err := Analyze(writeZipFile(t, map[string]string{
		".minecraft/mods/foo.jar":                "",
		".minecraft/versions/1.20.1/1.20.1.json": `{"id":"1.20.1"}`,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Format != model.FormatPlain || p.MCVersion != "1.20.1" || p.Stats.LocalJars != 1 {
		t.Fatalf("pack = %+v", p)
	}
}
