package pack

import (
	"archive/zip"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

type hmclModpack struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	GameVersion string `json:"gameVersion"`
	Description string `json:"description"`
}

type hmclPackJSON struct {
	ID           string `json:"id"`
	InheritsFrom string `json:"inheritsFrom"`
}

func parseHMCL(p *model.Pack, zr *zip.Reader) error {
	m, err := readJSON[hmclModpack](zr, p.Base+"modpack.json")
	if err != nil {
		return err
	}
	p.Name = m.Name
	p.Author = m.Author
	p.Version = m.Version
	p.MCVersion = m.GameVersion
	p.Description = m.Description

	// pack.json is a game-instance manifest. It can refine the game version but
	// does not contain the remote mod list.
	if pack, err := readJSON[hmclPackJSON](zr, p.Base+"minecraft/pack.json"); err == nil {
		if p.MCVersion == "" {
			if pack.InheritsFrom != "" {
				p.MCVersion = pack.InheritsFrom
			} else {
				p.MCVersion = pack.ID
			}
		}
	}
	if p.MCVersion == "" {
		addWarning(p, "HMCL modpack.json has no gameVersion")
	}

	addLocalJars(p, zr, p.Base+"minecraft", p.Base+".minecraft")
	return nil
}
