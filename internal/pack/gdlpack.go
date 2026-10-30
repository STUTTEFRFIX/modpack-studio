package pack

import (
	"archive/zip"
	"fmt"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

type gdlHashes struct {
	SHA512  string `json:"sha512"`
	SHA1    string `json:"sha1"`
	Murmur2 int64  `json:"murmur2"`
}

func (h gdlHashes) toMap() map[string]string {
	m := map[string]string{}
	if h.SHA512 != "" {
		m["sha512"] = h.SHA512
	}
	if h.SHA1 != "" {
		m["sha1"] = h.SHA1
	}
	if h.Murmur2 != 0 {
		m["murmur2"] = fmt.Sprintf("%d", h.Murmur2)
	}
	return m
}

type gdlPack struct {
	FormatVersion int    `json:"formatVersion"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Summary       string `json:"summary"`
	Author        string `json:"author"`
	Dependencies  struct {
		Minecraft  string `json:"minecraft"`
		Modloaders []struct {
			Type    string `json:"type"`
			Version string `json:"version"`
			Primary bool   `json:"primary"`
		} `json:"modloaders"`
	} `json:"dependencies"`
	Entries []struct {
		Type        string    `json:"type"`
		Hashes      gdlHashes `json:"hashes"`
		Description string    `json:"description"`
		Platforms   []struct {
			Hashes gdlHashes `json:"hashes"`
		} `json:"platforms"`
		OverridePaths []string `json:"overridePaths"`
	} `json:"entries"`
}

func parseGDLPack(p *model.Pack, zr *zip.Reader) error {
	m, err := readJSON[gdlPack](zr, p.Base+"gdlpack.json")
	if err != nil {
		return err
	}
	if m.FormatVersion != 1 {
		addWarning(p, fmt.Sprintf("gdlpack formatVersion=%d; expected 1", m.FormatVersion))
	}
	p.Name = m.Name
	p.Version = m.Version
	p.Description = m.Summary
	p.Author = m.Author
	p.MCVersion = m.Dependencies.Minecraft

	chosen := -1
	for i, l := range m.Dependencies.Modloaders {
		if l.Primary {
			chosen = i
			break
		}
	}
	if chosen < 0 && len(m.Dependencies.Modloaders) > 0 {
		chosen = 0
	}
	if chosen >= 0 {
		l := m.Dependencies.Modloaders[chosen]
		p.Loader = model.Loader{Type: normalizeLoaderType(l.Type), Version: l.Version}
	}

	for i, e := range m.Entries {
		switch e.Type {
		case "platform":
			hashes := e.Hashes.toMap()
			if len(hashes) == 0 {
				addWarning(p, "gdlpack platform entry has no hashes")
				continue
			}
			p.Mods = append(p.Mods, model.Mod{
				ID:      "gdl:" + firstHash(hashes),
				Kind:    "entry",
				Name:    fmt.Sprintf("platform entry %d", i+1),
				Enabled: true,
				Source:  model.Source{Kind: "gdlpack"},
				Hashes:  hashes,
			})
		case "optional":
			name := e.Description
			if name == "" {
				name = fmt.Sprintf("optional entry %d", i+1)
			}
			hashes := e.Hashes.toMap()
			if len(hashes) == 0 && len(e.Platforms) > 0 {
				hashes = e.Platforms[0].Hashes.toMap()
			}
			p.Mods = append(p.Mods, model.Mod{
				ID:       "gdl:optional:" + name + ":" + itoa(i),
				Kind:     "entry",
				Name:     name,
				Optional: true,
				Enabled:  true,
				Source:   model.Source{Kind: "gdlpack"},
				Hashes:   hashes,
				Tags:     e.OverridePaths,
			})
		default:
			addWarning(p, fmt.Sprintf("gdlpack entry type %q is not supported", e.Type))
		}
	}
	if len(m.Entries) > 0 {
		addWarning(p, "gdlpack entries are hash-based; resolve them against Modrinth/CurseForge before download")
	}

	addLocalJars(p, zr,
		p.Base+"overrides",
		p.Base+"clientOverrides",
		p.Base+"serverOverrides",
	)
	return nil
}

func firstHash(hashes map[string]string) string {
	for _, key := range []string{"sha512", "sha1", "murmur2"} {
		if v := hashes[key]; v != "" {
			return v
		}
	}
	return "unknown"
}
