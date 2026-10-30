package pack

import (
	"archive/zip"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

type serverManifest struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Description string `json:"description"`
	FileAPI     string `json:"fileApi"`
	Files       []struct {
		Path        string `json:"path"`
		Hash        string `json:"hash"`
		DownloadURL string `json:"downloadURL"`
	} `json:"files"`
	Addons []struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"addons"`
}

func parseServer(p *model.Pack, zr *zip.Reader) error {
	m, err := readJSON[serverManifest](zr, p.Base+"server-manifest.json")
	if err != nil {
		return err
	}
	p.Name = m.Name
	p.Author = m.Author
	p.Version = m.Version
	p.Description = m.Description

	for _, a := range m.Addons {
		switch a.ID {
		case "game":
			p.MCVersion = a.Version
		case "forge", "neoforge", "fabric", "quilt", "liteloader", "cleanroom":
			if p.Loader.Type == "" {
				p.Loader = model.Loader{Type: normalizeLoaderType(a.ID), Version: a.Version}
			}
		}
	}
	if p.MCVersion == "" {
		addWarning(p, "server-manifest.json addons do not declare a game version")
	}

	fileAPI := strings.TrimRight(m.FileAPI, "/")
	for _, f := range m.Files {
		if f.Path == "" {
			addWarning(p, "server-manifest.json has a file entry with an empty path")
			continue
		}
		url := f.DownloadURL
		if url == "" {
			if fileAPI == "" {
				addWarning(p, "server-manifest.json fileApi is empty; file downloads cannot be resolved")
			} else {
				url = fileAPI + "/overrides/" + f.Path
			}
		}
		p.Mods = append(p.Mods, model.Mod{
			ID:      "server:" + f.Path,
			Kind:    "remote",
			Path:    f.Path,
			Name:    f.Path,
			Enabled: true,
			Source:  model.Source{Kind: "server", URL: url},
			Hashes:  map[string]string{"sha1": f.Hash},
		})
	}
	addLocalJars(p, zr, p.Base+"overrides", p.Base+"mods")
	return nil
}
