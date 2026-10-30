// Package pack detects modpack formats and parses their manifests into the
// format-neutral model.
package pack

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

// Analyze opens a pack archive, detects its format and parses the manifest.
func Analyze(path string) (*model.Pack, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer zr.Close()

	res := Sniff(&zr.Reader)
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	p := &model.Pack{
		SourcePath: path,
		Format:     res.Format,
		Base:       res.Base,
		Name:       name,
	}

	var perr error
	switch res.Format {
	case model.FormatCurseForge:
		perr = parseCurseForge(p, &zr.Reader)
	case model.FormatModrinth:
		perr = parseModrinth(p, &zr.Reader)
	case model.FormatMCBBS:
		perr = parseMCBBS(p, &zr.Reader)
	case model.FormatMultiMC:
		perr = parseMultiMC(p, &zr.Reader)
	case model.FormatHMCL:
		perr = parseHMCL(p, &zr.Reader)
	case model.FormatServer:
		perr = parseServer(p, &zr.Reader)
	case model.FormatGDLPack:
		perr = parseGDLPack(p, &zr.Reader)
	case model.FormatPackwiz:
		perr = parsePackwiz(p, &zr.Reader)
	case model.FormatFTBApp:
		perr = parseFTBApp(p, &zr.Reader)
	case model.FormatATLauncher:
		perr = parseATLauncher(p, &zr.Reader)
	case model.FormatTechnic:
		perr = parseTechnic(p, &zr.Reader)
	case model.FormatLauncherBundle:
		perr = parseLauncherBundle(p, &zr.Reader)
	case model.FormatPlain, model.FormatUnknown:
		perr = parsePlain(p, &zr.Reader)
	default:
		addWarning(p, fmt.Sprintf("format %q has no parser", res.Format))
	}
	if perr != nil {
		return nil, fmt.Errorf("%s: %w", res.Format, perr)
	}

	finalize(p)
	return p, nil
}
