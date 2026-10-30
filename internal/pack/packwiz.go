package pack

import (
	"archive/zip"
	"regexp"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

var (
	tomlString = func(key string) *regexp.Regexp {
		return regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*"([^"]*)"`)
	}
	packwizName   = tomlString("name")
	packwizAuth   = tomlString("author")
	packwizVer    = tomlString("version")
	packwizDesc   = tomlString("description")
	packwizMC     = tomlString("minecraft")
	packwizForge  = tomlString("forge")
	packwizFabric = tomlString("fabric")
	packwizQuilt  = tomlString("quilt")
	packwizNeo    = tomlString("neoforge")
)

func parsePackwiz(p *model.Pack, zr *zip.Reader) error {
	data, err := readFile(zr, p.Base+"pack.toml")
	if err != nil {
		return err
	}
	text := string(data)
	if v := packwizName.FindStringSubmatch(text); v != nil {
		p.Name = v[1]
	}
	if v := packwizAuth.FindStringSubmatch(text); v != nil {
		p.Author = v[1]
	}
	if v := packwizVer.FindStringSubmatch(text); v != nil {
		p.Version = v[1]
	}
	if v := packwizDesc.FindStringSubmatch(text); v != nil {
		p.Description = v[1]
	}
	if v := packwizMC.FindStringSubmatch(text); v != nil {
		p.MCVersion = v[1]
	}

	// pack.toml has a [versions] section. A flat regex is enough for the
	// well-known loader keys; full TOML parsing arrives with the index reader.
	versions := extractTOMLSection(text, "versions")
	if versions != "" {
		for _, item := range []struct {
			key string
			re  *regexp.Regexp
			typ string
		}{
			{"forge", packwizForge, "forge"},
			{"neoforge", packwizNeo, "neoforge"},
			{"fabric", packwizFabric, "fabric"},
			{"quilt", packwizQuilt, "quilt"},
		} {
			if v := item.re.FindStringSubmatch(versions); v != nil {
				p.Loader = model.Loader{Type: item.typ, Version: v[1]}
				break
			}
		}
	}

	addWarning(p, "packwiz: index.toml and *.pw.toml are not parsed yet; only pack.toml metadata is available")
	addLocalJars(p, zr, p.Base+"mods")
	return nil
}

func extractTOMLSection(text, section string) string {
	lines := strings.Split(text, "\n")
	var out []string
	in := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
			in = name == section
			continue
		}
		if in {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
