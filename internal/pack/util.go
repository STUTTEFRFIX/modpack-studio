package pack

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func trimBOM(b []byte) []byte { return bytes.TrimPrefix(b, utf8BOM) }

func readFile(zr *zip.Reader, name string) ([]byte, error) {
	f, err := zr.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return trimBOM(b), nil
}

func readJSON[T any](zr *zip.Reader, name string) (T, error) {
	var v T
	b, err := readFile(zr, name)
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return v, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func entryNames(zr *zip.Reader) map[string]*zip.File {
	m := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		m[f.Name] = f
	}
	return m
}

// candidateBases returns the archive root and every immediate subdirectory,
// in archive order. Packs are often wrapped in one extra folder.
func candidateBases(zr *zip.Reader) []string {
	bases := []string{""}
	seen := map[string]bool{"": true}
	for _, f := range zr.File {
		name := strings.TrimSuffix(f.Name, "/")
		i := strings.IndexByte(name, '/')
		if i <= 0 || i == len(name)-1 {
			continue
		}
		base := name[:i+1]
		if !seen[base] {
			seen[base] = true
			bases = append(bases, base)
		}
	}
	return bases
}

func normalizePrefix(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\\", "/"))
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimSuffix(s, "/")
	if s == "" {
		return ""
	}
	return s + "/"
}

func addWarning(p *model.Pack, msg string) {
	if msg == "" {
		return
	}
	for _, w := range p.Warnings {
		if w == msg {
			return
		}
	}
	p.Warnings = append(p.Warnings, msg)
}

// addLocalJars scans direct children of <dir>/mods/ for .jar files. It is
// deliberately shallow: nested jars are libs or jar-in-jar payloads and should
// not be treated as pack-level mods.
func addLocalJars(p *model.Pack, zr *zip.Reader, dirs ...string) {
	seen := map[string]bool{}
	for _, dir := range dirs {
		prefix := normalizePrefix(dir) + "mods/"
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			name := f.Name
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			rest := strings.TrimPrefix(name, prefix)
			if rest == "" || strings.Contains(rest, "/") {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(rest), ".jar") {
				continue
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			p.Mods = append(p.Mods, model.Mod{
				ID:      "local:" + name,
				Kind:    "local",
				Path:    name,
				Name:    path.Base(rest),
				Enabled: true,
				Source:  model.Source{Kind: "local"},
			})
		}
	}
}

func finalize(p *model.Pack) {
	p.Stats = model.Stats{}
	for _, m := range p.Mods {
		switch m.Kind {
		case "remote":
			p.Stats.RemoteFiles++
		case "local":
			p.Stats.LocalJars++
		case "entry":
			p.Stats.Entries++
		}
		if m.Optional {
			p.Stats.Optional++
		}
	}
	p.Stats.TotalMods = len(p.Mods)
}

func normalizeLoaderType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "fabric-loader", "fabric":
		return "fabric"
	case "quilt-loader", "quilt":
		return "quilt"
	case "neo-forge", "neoforge":
		return "neoforge"
	case "forge":
		return "forge"
	case "liteloader":
		return "liteloader"
	case "cleanroom":
		return "cleanroom"
	case "legacyfabric", "legacy-fabric":
		return "legacyfabric"
	default:
		return s
	}
}

// splitLoaderID splits ids like "forge-47.2.0" or "neoforge-1.20.1-47.1.106".
func splitLoaderID(id string) (typ, version string) {
	id = strings.TrimSpace(id)
	i := strings.IndexByte(id, '-')
	if i < 0 {
		return normalizeLoaderType(id), ""
	}
	return normalizeLoaderType(id[:i]), id[i+1:]
}

func stripMCVersionPrefix(version, mc string) string {
	if mc == "" {
		return version
	}
	prefix := mc + "-"
	if strings.HasPrefix(version, prefix) {
		return strings.TrimPrefix(version, prefix)
	}
	return version
}

func itoa(i int) string { return strconv.Itoa(i) }
