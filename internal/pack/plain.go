package pack

import (
	"archive/zip"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

type plainVersionJSON struct {
	ID           string `json:"id"`
	InheritsFrom string `json:"inheritsFrom"`
}

func parsePlain(p *model.Pack, zr *zip.Reader) error {
	// Try to read versions/<id>/<id>.json for a game version hint.
	prefix := p.Base + "versions/"
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) || !strings.HasSuffix(f.Name, ".json") {
			continue
		}
		rest := strings.TrimPrefix(f.Name, prefix)
		if strings.Count(rest, "/") != 1 {
			continue
		}
		if v, err := readJSON[plainVersionJSON](zr, f.Name); err == nil {
			if v.InheritsFrom != "" {
				p.MCVersion = v.InheritsFrom
			} else if v.ID != "" {
				p.MCVersion = v.ID
			}
			break
		}
	}
	if p.MCVersion == "" {
		addWarning(p, "plain ZIP: game version could not be inferred")
	}

	addLocalJars(p, zr, p.Base)
	return nil
}
