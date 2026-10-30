package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/STUTTEFRFIX/modpack-studio/internal/model"
	"github.com/STUTTEFRFIX/modpack-studio/internal/pack"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "analyze":
		cmdAnalyze(os.Args[2:])
	case "sniff":
		cmdSniff(os.Args[2:])
	case "version", "-version", "--version":
		fmt.Println("modpack-studio", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `modpack-studio - Minecraft modpack analysis toolkit

Usage:
  modpack-studio analyze <pack.zip|pack.mrpack> [--json] [--compact]
  modpack-studio sniff   <pack.zip|pack.mrpack>
  modpack-studio version

Examples:
  modpack-studio analyze create-live-5.mrpack
  modpack-studio analyze create-live-5.mrpack --json
  modpack-studio sniff create-live-5.mrpack
`)
}

func cmdAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "print the normalized pack as JSON")
	compact := fs.Bool("compact", false, "compact JSON output")

	// Accept both "analyze --json pack.zip" and "analyze pack.zip --json".
	var flags, positional []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
		} else {
			positional = append(positional, a)
		}
	}
	if err := fs.Parse(flags); err != nil {
		os.Exit(2)
	}
	if len(positional) != 1 {
		fmt.Fprintln(os.Stderr, "analyze: expected exactly one pack path")
		os.Exit(2)
	}

	p, err := pack.Analyze(positional[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		if !*compact {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(p); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}
	printSummary(p)
}

func printSummary(p *model.Pack) {
	fmt.Printf("Format:    %s (%s)\n", p.Format, p.Format.Label())
	fmt.Printf("Name:      %s\n", p.Name)
	if p.Version != "" {
		fmt.Printf("Version:   %s\n", p.Version)
	}
	if p.MCVersion != "" {
		fmt.Printf("Minecraft: %s\n", p.MCVersion)
	}
	if p.Loader.Type != "" {
		fmt.Printf("Loader:    %s %s\n", p.Loader.Type, p.Loader.Version)
	}
	fmt.Printf("Mods:      %d total, %d remote, %d local, %d entries, %d optional\n",
		p.Stats.TotalMods, p.Stats.RemoteFiles, p.Stats.LocalJars, p.Stats.Entries, p.Stats.Optional)
	if p.Base != "" {
		fmt.Printf("Base:      %s\n", p.Base)
	}
	for _, w := range p.Warnings {
		fmt.Printf("Warning:   %s\n", w)
	}
}

func cmdSniff(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "sniff: expected exactly one pack path")
		os.Exit(2)
	}
	zr, err := zip.OpenReader(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer zr.Close()
	res := pack.Sniff(&zr.Reader)
	fmt.Printf("%s\tbase=%q\n", res.Format, res.Base)
}
