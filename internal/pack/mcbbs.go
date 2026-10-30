package pack

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

type mcbbsManifest struct {
	ManifestType    string `json:"manifestType"`
	ManifestVersion int    `json:"manifestVersion"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Author          string `json:"author"`
	Description     string `json:"description"`
	FileAPI         string `json:"fileApi"`
	URL             string `json:"url"`
	ForceUpdate     bool   `json:"forceUpdate"`
	Origin          []struct {
		Type string `json:"type"`
		ID   int    `json:"id"`
	} `json:"origin"`
	Addons []struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"addons"`
	Files []struct {
		Type      string `json:"type"`
		Path      string `json:"path"`
		Hash      string `json:"hash"`
		ProjectID int    `json:"projectID"`
		FileID    int    `json:"fileID"`
		FileName  string `json:"fileName"`
		URL       string `json:"url"`
		Force     bool   `json:"force"`
	} `json:"files"`
}

func parseMCBBS(p *model.Pack, zr *zip.Reader) error {
	manifestName := p.Base + "mcbbs.packmeta"
	names := entryNames(zr)
	if _, ok := names[manifestName]; !ok {
		manifestName = p.Base + "manifest.json"
	}
	m, err := readJSON[mcbbsManifest](zr, manifestName)
	if err != nil {
		return err
	}
	if m.ManifestType != "minecraftModpack" {
		return fmt.Errorf("%s: manifestType is %q, want minecraftModpack", manifestName, m.ManifestType)
	}
	if len(m.Addons) == 0 {
		return fmt.Errorf("%s: MCBBS manifest has no addons", manifestName)
	}

	p.Name = m.Name
	p.Version = m.Version
	p.Author = m.Author
	p.Description = m.Description

	for _, a := range m.Addons {
		switch a.ID {
		case "game":
			p.MCVersion = a.Version
		case "forge", "neoforge", "fabric", "quilt", "liteloader", "cleanroom":
			if p.Loader.Type == "" {
				p.Loader = model.Loader{Type: normalizeLoaderType(a.ID), Version: a.Version}
			}
		case "optifine":
			addWarning(p, "MCBBS addon optifine is present; loader resolution is not implemented yet")
		}
	}
	if p.MCVersion == "" {
		addWarning(p, "MCBBS addons do not declare a game version")
	}

	fileAPI := strings.TrimRight(m.FileAPI, "/")
	for _, f := range m.Files {
		switch f.Type {
		case "addon":
			if f.Path == "" {
				addWarning(p, "MCBBS addon file entry has an empty path")
				continue
			}
			url := ""
			if fileAPI != "" {
				url = fileAPI + "/overrides/" + f.Path
			} else {
				addWarning(p, "MCBBS fileApi is empty; addon files cannot be downloaded")
			}
			p.Mods = append(p.Mods, model.Mod{
				ID:      "mcbbs:" + f.Path,
				Kind:    "remote",
				Path:    f.Path,
				Name:    f.Path,
				Enabled: true,
				Source:  model.Source{Kind: "mcbbs", URL: url},
				Hashes:  map[string]string{"sha1": f.Hash},
			})
		case "curse":
			name := f.FileName
			relPath := ""
			if name != "" {
				relPath = "mods/" + name
			} else {
				name = fmt.Sprintf("project %d / file %d", f.ProjectID, f.FileID)
			}
			p.Mods = append(p.Mods, model.Mod{
				ID:      "curseforge:" + itoa(f.ProjectID) + ":" + itoa(f.FileID),
				Kind:    "remote",
				Path:    relPath,
				Name:    name,
				Enabled: true,
				Source: model.Source{
					Kind:      "curseforge",
					ProjectID: itoa(f.ProjectID),
					FileID:    itoa(f.FileID),
					URL:       f.URL,
				},
			})
		default:
			addWarning(p, fmt.Sprintf("MCBBS file type %q is not supported", f.Type))
		}
	}

	addLocalJars(p, zr, p.Base+"overrides", p.Base+"mods")
	return nil
}
