// Package webui embeds the single-file workbench used by the desktop server
// and copied into the Android app at build time.
package webui

import _ "embed"

//go:embed graph.html
var GraphHTML []byte
