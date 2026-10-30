package pack

import (
	"archive/zip"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

type ftbInstance struct {
	Name      string `json:"name"`
	MCVersion string `json:"mcVersion"`
	ModLoader string `json:"modLoader"`
}

func parseFTBApp(p *model.Pack, zr *zip.Reader) error {
	m, err := readJSON[ftbInstance](zr, p.Base+"instance.json")
	if err != nil {
		return err
	}
	p.Name = m.Name
	p.MCVersion = m.MCVersion
	if m.ModLoader != "" {
		typ, version := splitLoaderID(m.ModLoader)
		if typ == "forge" || typ == "neoforge" {
			version = stripMCVersionPrefix(version, p.MCVersion)
		}
		p.Loader = model.Loader{Type: typ, Version: version}
	}
	addWarning(p, "FTB App: mod files are resolved from modpacks.ch; not implemented in this parser yet")
	addLocalJars(p, zr, p.Base+"mods", p.Base+"minecraft/mods")
	return nil
}

type atlInstance struct {
	InheritsFrom string `json:"inheritsFrom"`
	Launcher     struct {
		Name          string `json:"name"`
		Version       string `json:"version"`
		LoaderVersion struct {
			Version string `json:"version"`
			Type    string `json:"type"`
		} `json:"loaderVersion"`
	} `json:"launcher"`
}

func parseATLauncher(p *model.Pack, zr *zip.Reader) error {
	m, err := readJSON[atlInstance](zr, p.Base+"instance.json")
	if err != nil {
		return err
	}
	p.Name = m.Launcher.Name
	p.Version = m.Launcher.Version
	p.MCVersion = m.InheritsFrom
	if m.Launcher.LoaderVersion.Version != "" || m.Launcher.LoaderVersion.Type != "" {
		p.Loader = model.Loader{
			Type:    normalizeLoaderType(m.Launcher.LoaderVersion.Type),
			Version: m.Launcher.LoaderVersion.Version,
		}
	}
	addWarning(p, "ATLauncher: instance.json is an internal format; export to CurseForge/Modrinth/MultiMC for interchange")
	addLocalJars(p, zr, p.Base+"mods", p.Base+"minecraft/mods")
	return nil
}

type technicVersion struct {
	ID           string `json:"id"`
	InheritsFrom string `json:"inheritsFrom"`
}

func parseTechnic(p *model.Pack, zr *zip.Reader) error {
	if v, err := readJSON[technicVersion](zr, p.Base+"bin/version.json"); err == nil {
		p.MCVersion = v.InheritsFrom
		if p.MCVersion == "" {
			p.MCVersion = v.ID
		}
	}
	if p.MCVersion == "" {
		addWarning(p, "Technic: could not infer Minecraft version from bin/version.json")
	}
	addWarning(p, "Technic: bin/modpack.jar loader resolution and Solder builds are not implemented yet")
	addLocalJars(p, zr, p.Base)
	return nil
}

func parseLauncherBundle(p *model.Pack, zr *zip.Reader) error {
	addWarning(p, "launcher bundle: contains modpack.zip/modpack.mrpack; extract and re-analyze the inner pack")
	return nil
}
