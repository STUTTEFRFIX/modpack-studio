package pack

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

type curseManifest struct {
	ManifestType    string `json:"manifestType"`
	ManifestVersion int    `json:"manifestVersion"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Author          string `json:"author"`
	Description     string `json:"description"`
	WebsiteURL      string `json:"websiteUrl"`
	ProjectID       int    `json:"projectID"`
	FileID          int    `json:"fileID"`
	Minecraft       struct {
		Version    string `json:"version"`
		ModLoaders []struct {
			ID      string `json:"id"`
			Primary bool   `json:"primary"`
		} `json:"modLoaders"`
	} `json:"minecraft"`
	Files []struct {
		ProjectID int               `json:"projectID"`
		FileID    int               `json:"fileID"`
		Required  *bool             `json:"required"`
		FileName  string            `json:"fileName"`
		URL       string            `json:"url"`
		Hashes    map[string]string `json:"hashes"`
	} `json:"files"`
	Overrides string `json:"overrides"`
}

func parseCurseForge(p *model.Pack, zr *zip.Reader) error {
	m, err := readJSON[curseManifest](zr, p.Base+"manifest.json")
	if err != nil {
		return err
	}
	if m.ManifestType != "minecraftModpack" {
		return fmt.Errorf("manifest.json: manifestType is %q, want minecraftModpack", m.ManifestType)
	}
	if m.ManifestVersion != 1 {
		addWarning(p, fmt.Sprintf("CurseForge manifestVersion=%d; expected 1", m.ManifestVersion))
	}

	p.Name = m.Name
	p.Version = m.Version
	p.Author = m.Author
	p.Description = m.Description
	p.MCVersion = m.Minecraft.Version

	// Pick the primary loader, otherwise the first one.
	chosen := -1
	for i, l := range m.Minecraft.ModLoaders {
		if l.Primary {
			chosen = i
			break
		}
	}
	if chosen < 0 && len(m.Minecraft.ModLoaders) > 0 {
		chosen = 0
	}
	if chosen >= 0 {
		typ, version := splitLoaderID(m.Minecraft.ModLoaders[chosen].ID)
		if typ == "forge" || typ == "neoforge" {
			version = stripMCVersionPrefix(version, p.MCVersion)
		}
		p.Loader = model.Loader{Type: typ, Version: version}
	}
	if len(m.Minecraft.ModLoaders) > 1 {
		addWarning(p, "CurseForge manifest declares multiple mod loaders; only the primary one is kept")
	}

	for _, f := range m.Files {
		required := true
		if f.Required != nil {
			required = *f.Required
		}
		name := f.FileName
		relPath := ""
		if name != "" {
			relPath = "mods/" + name
		} else {
			name = fmt.Sprintf("project %d / file %d", f.ProjectID, f.FileID)
			addWarning(p, "CurseForge entries omit fileName; resolve names via the CurseForge API/CDN")
		}
		p.Mods = append(p.Mods, model.Mod{
			ID:       "curseforge:" + itoa(f.ProjectID) + ":" + itoa(f.FileID),
			Kind:     "remote",
			Path:     relPath,
			Name:     name,
			Optional: !required,
			Enabled:  true,
			Source: model.Source{
				Kind:      "curseforge",
				ProjectID: itoa(f.ProjectID),
				FileID:    itoa(f.FileID),
				URL:       f.URL,
			},
			Hashes: f.Hashes,
		})
	}
	overrides := m.Overrides
	if strings.TrimSpace(overrides) == "" {
		overrides = "overrides"
	}
	addLocalJars(p, zr, p.Base+overrides, p.Base+"mods")
	return nil
}
