// Package model contains the format-neutral data model shared by all pack
// parsers, the CLI and (later) the HTTP API.
package model

import "fmt"

// Format is a detected modpack format.
type Format string

const (
	FormatCurseForge     Format = "curseforge"
	FormatModrinth       Format = "modrinth"
	FormatMCBBS          Format = "mcbbs"
	FormatMultiMC        Format = "multimc"
	FormatHMCL           Format = "hmcl"
	FormatServer         Format = "server"
	FormatGDLPack        Format = "gdlpack"
	FormatPackwiz        Format = "packwiz"
	FormatFTBApp         Format = "ftb-app"
	FormatATLauncher     Format = "atlauncher"
	FormatTechnic        Format = "technic"
	FormatLauncherBundle Format = "launcher-bundle"
	FormatPlain          Format = "plain"
	FormatUnknown        Format = "unknown"
)

// Label returns a human-readable format name.
func (f Format) Label() string {
	switch f {
	case FormatCurseForge:
		return "CurseForge manifest.json"
	case FormatModrinth:
		return "Modrinth .mrpack"
	case FormatMCBBS:
		return "MCBBS / mcbbs.packmeta"
	case FormatMultiMC:
		return "MultiMC / Prism instance"
	case FormatHMCL:
		return "HMCL native"
	case FormatServer:
		return "HMCL server pack"
	case FormatGDLPack:
		return "GDLauncher .gdlpack"
	case FormatPackwiz:
		return "packwiz"
	case FormatFTBApp:
		return "FTB App instance"
	case FormatATLauncher:
		return "ATLauncher instance"
	case FormatTechnic:
		return "Technic / Solder"
	case FormatLauncherBundle:
		return "launcher bundle"
	case FormatPlain:
		return "plain ZIP"
	default:
		return string(f)
	}
}

// Pack is the normalized view of a modpack.
type Pack struct {
	SourcePath  string   `json:"sourcePath"`
	Format      Format   `json:"format"`
	Base        string   `json:"base,omitempty"`
	Name        string   `json:"name,omitempty"`
	Version     string   `json:"version,omitempty"`
	Author      string   `json:"author,omitempty"`
	Description string   `json:"description,omitempty"`
	MCVersion   string   `json:"mcVersion,omitempty"`
	Loader      Loader   `json:"loader"`
	Mods        []Mod    `json:"mods"`
	Warnings    []string `json:"warnings,omitempty"`
	Stats       Stats    `json:"stats"`
}

// Loader describes the mod loader required by the pack.
type Loader struct {
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

// Mod is either a remote file reference or a local jar found in the archive.
type Mod struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"` // remote | local | entry
	Path         string            `json:"path,omitempty"`
	Name         string            `json:"name,omitempty"`
	Version      string            `json:"version,omitempty"`
	ModID        string            `json:"modid,omitempty"`
	Side         string            `json:"side,omitempty"` // client | server | both
	Optional     bool              `json:"optional,omitempty"`
	Enabled      bool              `json:"enabled"`
	Source       Source            `json:"source"`
	Hashes       map[string]string `json:"hashes,omitempty"`
	FileSize     int64             `json:"fileSize,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Dependencies []Dependency      `json:"dependencies,omitempty"`
}

// Source identifies where a file came from.
type Source struct {
	Kind      string `json:"kind"` // curseforge | modrinth | mcbbs | server | local | gdlpack | ...
	ProjectID string `json:"projectId,omitempty"`
	FileID    string `json:"fileId,omitempty"`
	VersionID string `json:"versionId,omitempty"`
	URL       string `json:"url,omitempty"`
}

// Dependency is a resolved or unresolved relation. The graph layer fills these
// in later; parsers currently only keep what the manifest directly states.
type Dependency struct {
	Target string `json:"target,omitempty"`
	Type   string `json:"type,omitempty"` // required | optional | incompatible | embedded
	Range  string `json:"range,omitempty"`
}

// Stats is a small summary used by the CLI and the graph UI.
type Stats struct {
	TotalMods   int `json:"totalMods"`
	RemoteFiles int `json:"remoteFiles"`
	LocalJars   int `json:"localJars"`
	Entries     int `json:"entries"`
	Optional    int `json:"optional"`
}

// Summary is a convenience string for CLI output.
func (p *Pack) Summary() string {
	loader := p.Loader.Type
	if p.Loader.Version != "" {
		loader += " " + p.Loader.Version
	}
	if loader == "" {
		loader = "unknown loader"
	}
	return fmt.Sprintf("%s | %s | %s | %d mods (%d remote, %d local)",
		p.Name, p.MCVersion, loader, p.Stats.TotalMods, p.Stats.RemoteFiles, p.Stats.LocalJars)
}
