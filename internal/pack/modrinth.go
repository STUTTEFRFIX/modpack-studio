package pack

import (
	"archive/zip"
	"fmt"
	"path"
	"regexp"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

var modrinthURLPattern = regexp.MustCompile(`/data/([^/]+)/versions/([^/]+)/`)

type modrinthIndex struct {
	FormatVersion int    `json:"formatVersion"`
	Game          string `json:"game"`
	VersionID     string `json:"versionId"`
	Name          string `json:"name"`
	Summary       string `json:"summary"`
	Files         []struct {
		Path      string            `json:"path"`
		Hashes    map[string]string `json:"hashes"`
		Env       map[string]string `json:"env"`
		Downloads []string          `json:"downloads"`
		FileSize  int64             `json:"fileSize"`
	} `json:"files"`
	Dependencies map[string]string `json:"dependencies"`
}

func parseModrinth(p *model.Pack, zr *zip.Reader) error {
	m, err := readJSON[modrinthIndex](zr, p.Base+"modrinth.index.json")
	if err != nil {
		return err
	}
	if m.FormatVersion != 1 {
		addWarning(p, fmt.Sprintf("Modrinth formatVersion=%d; this parser targets 1", m.FormatVersion))
	}
	if m.Game != "" && m.Game != "minecraft" {
		addWarning(p, fmt.Sprintf("Modrinth index game=%q; expected minecraft", m.Game))
	}

	p.Name = m.Name
	p.Version = m.VersionID
	p.Description = m.Summary
	p.MCVersion = m.Dependencies["minecraft"]

	for _, key := range []string{"forge", "neoforge", "fabric-loader", "quilt-loader"} {
		if version, ok := m.Dependencies[key]; ok && version != "" {
			p.Loader = model.Loader{Type: normalizeLoaderType(key), Version: version}
			break
		}
	}
	if p.Loader.Type == "" && len(m.Dependencies) > 0 {
		for key := range m.Dependencies {
			if key != "minecraft" {
				addWarning(p, fmt.Sprintf("Modrinth dependency %q has no known loader mapping", key))
				break
			}
		}
	}

	for _, f := range m.Files {
		client := f.Env["client"]
		server := f.Env["server"]
		optional := client == "optional" || server == "optional"
		side := "both"
		switch {
		case client == "unsupported" && server == "unsupported":
			side = "none"
		case client == "unsupported":
			side = "server"
		case server == "unsupported":
			side = "client"
		}

		projectID, versionID := "", ""
		if len(f.Downloads) > 0 {
			if mm := modrinthURLPattern.FindStringSubmatch(f.Downloads[0]); mm != nil {
				projectID, versionID = mm[1], mm[2]
			}
		}
		if len(f.Downloads) == 0 {
			addWarning(p, "Modrinth entry has no download URL")
		}
		if f.Hashes["sha1"] == "" || f.Hashes["sha512"] == "" {
			addWarning(p, "Modrinth entry is missing sha1 and/or sha512")
		}

		id := "modrinth:" + f.Path
		if projectID != "" {
			id = "modrinth:" + projectID + ":" + versionID
		}
		url := ""
		if len(f.Downloads) > 0 {
			url = f.Downloads[0]
		}
		p.Mods = append(p.Mods, model.Mod{
			ID:       id,
			Kind:     "remote",
			Path:     f.Path,
			Name:     path.Base(f.Path),
			Side:     side,
			Optional: optional,
			Enabled:  true,
			Source: model.Source{
				Kind:      "modrinth",
				ProjectID: projectID,
				VersionID: versionID,
				URL:       url,
			},
			Hashes:   f.Hashes,
			FileSize: f.FileSize,
		})
	}

	addLocalJars(p, zr,
		p.Base+"overrides",
		p.Base+"client-overrides",
		p.Base+"server-overrides",
		p.Base+"mods",
	)
	return nil
}
