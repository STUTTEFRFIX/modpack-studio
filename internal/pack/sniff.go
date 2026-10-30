package pack

import (
	"archive/zip"
	"encoding/json"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

// SniffResult is the detected format plus the archive-relative base directory.
type SniffResult struct {
	Format model.Format
	Base   string
}

// Sniff identifies a pack from its ZIP entries. It looks at the archive root
// and every immediate subdirectory, and uses the same priority order as the
// launchers documented in docs/Minecraft整合包格式全解.md.
func Sniff(zr *zip.Reader) SniffResult {
	names := entryNames(zr)
	for _, base := range candidateBases(zr) {
		has := func(n string) bool {
			_, ok := names[base+n]
			return ok
		}

		if has("mcbbs.packmeta") {
			return SniffResult{model.FormatMCBBS, base}
		}
		if has("mmc-pack.json") && has("instance.cfg") {
			return SniffResult{model.FormatMultiMC, base}
		}
		if has("modrinth.index.json") {
			return SniffResult{model.FormatModrinth, base}
		}
		if has("manifest.json") {
			if b, err := readFile(zr, base+"manifest.json"); err == nil {
				var probe struct {
					Addons       json.RawMessage `json:"addons"`
					ManifestType string          `json:"manifestType"`
				}
				if json.Unmarshal(b, &probe) == nil {
					if len(probe.Addons) > 0 && string(probe.Addons) != "null" {
						return SniffResult{model.FormatMCBBS, base}
					}
					if probe.ManifestType == "minecraftModpack" {
						return SniffResult{model.FormatCurseForge, base}
					}
				}
			}
		}
		if has("modpack.json") && has("minecraft/pack.json") {
			return SniffResult{model.FormatHMCL, base}
		}
		if has("server-manifest.json") {
			return SniffResult{model.FormatServer, base}
		}
		if has("pack.toml") {
			return SniffResult{model.FormatPackwiz, base}
		}
		if has("gdlpack.json") {
			return SniffResult{model.FormatGDLPack, base}
		}
		if has("instance.json") {
			if b, err := readFile(zr, base+"instance.json"); err == nil {
				var probe map[string]json.RawMessage
				if json.Unmarshal(b, &probe) == nil {
					_, hasUUID := probe["uuid"]
					_, hasVersionID := probe["versionId"]
					_, hasMC := probe["mcVersion"]
					if hasUUID && hasVersionID && hasMC {
						return SniffResult{model.FormatFTBApp, base}
					}
					if launcher, ok := probe["launcher"]; ok && len(launcher) > 0 && string(launcher) != "null" {
						return SniffResult{model.FormatATLauncher, base}
					}
				}
			}
		}
		if has("bin/modpack.jar") || has("bin/version.json") {
			return SniffResult{model.FormatTechnic, base}
		}
		if has("modpack.zip") || has("modpack.mrpack") {
			return SniffResult{model.FormatLauncherBundle, base}
		}
	}

	// Plain ZIP: infer the game directory from common layouts.
	for _, base := range []string{".minecraft/", "minecraft/", ""} {
		prefix := base + "mods/"
		for name := range names {
			if strings.HasPrefix(name, prefix) {
				return SniffResult{model.FormatPlain, base}
			}
		}
	}
	for name := range names {
		if strings.HasPrefix(name, "versions/") {
			return SniffResult{model.FormatPlain, ""}
		}
	}
	return SniffResult{model.FormatPlain, ""}
}
