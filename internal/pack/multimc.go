package pack

import (
	"archive/zip"
	"bufio"
	"fmt"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
)

type mmcPack struct {
	FormatVersion int `json:"formatVersion"`
	Components    []struct {
		UID            string `json:"uid"`
		Version        string `json:"version"`
		Important      bool   `json:"important"`
		DependencyOnly bool   `json:"dependencyOnly"`
	} `json:"components"`
}

func parseINI(data []byte) map[string]string {
	out := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "[") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

func parseMultiMC(p *model.Pack, zr *zip.Reader) error {
	cfgData, err := readFile(zr, p.Base+"instance.cfg")
	if err != nil {
		return err
	}
	cfg := parseINI(cfgData)
	p.Name = cfg["name"]
	p.Description = cfg["notes"]
	if it := cfg["InstanceType"]; it != "" && it != "OneSix" {
		addWarning(p, fmt.Sprintf("MultiMC InstanceType=%q; this parser targets OneSix", it))
	}

	m, err := readJSON[mmcPack](zr, p.Base+"mmc-pack.json")
	if err != nil {
		return err
	}
	if m.FormatVersion != 1 {
		addWarning(p, fmt.Sprintf("mmc-pack.json formatVersion=%d; expected 1", m.FormatVersion))
	}
	for _, c := range m.Components {
		switch c.UID {
		case "net.minecraft":
			p.MCVersion = c.Version
		case "net.minecraftforge":
			if p.Loader.Type == "" {
				p.Loader = model.Loader{Type: "forge", Version: c.Version}
			}
		case "net.neoforged":
			if p.Loader.Type == "" {
				p.Loader = model.Loader{Type: "neoforge", Version: c.Version}
			}
		case "net.fabricmc.fabric-loader":
			if p.Loader.Type == "" {
				p.Loader = model.Loader{Type: "fabric", Version: c.Version}
			}
		case "org.quiltmc.quilt-loader":
			if p.Loader.Type == "" {
				p.Loader = model.Loader{Type: "quilt", Version: c.Version}
			}
		case "com.mumfrey.liteloader":
			if p.Loader.Type == "" {
				p.Loader = model.Loader{Type: "liteloader", Version: c.Version}
			}
		}
	}
	if p.MCVersion == "" {
		addWarning(p, "MultiMC mmc-pack.json has no net.minecraft component")
	}

	names := entryNames(zr)
	gameDir := p.Base + "minecraft"
	if _, ok := names[gameDir+"/"]; !ok {
		if _, ok := names[p.Base+".minecraft/"]; ok {
			gameDir = p.Base + ".minecraft"
		}
	}
	addLocalJars(p, zr, gameDir, p.Base+"minecraft", p.Base+".minecraft")
	return nil
}
