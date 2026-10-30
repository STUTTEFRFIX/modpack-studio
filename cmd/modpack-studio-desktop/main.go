// Command modpack-studio-desktop serves the embedded workbench on localhost and
// opens the default browser. The first build bundles the Create Live 5 demo.
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/STUTTEFRFIX/modpack-studio/internal/webui"
)

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func main() {
	noBrowser := flag.Bool("no-browser", false, "do not open a browser")
	port := flag.Int("port", 0, "listen port (0 = random)")
	flag.Parse()

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "listen:", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(webui.GraphHTML)
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "serve:", err)
		}
	}()

	url := "http://" + ln.Addr().String() + "/"
	fmt.Println("Modpack Studio 0.2.1-import")
	fmt.Println("URL:", url)
	fmt.Println("Press Ctrl+C to stop.")
	if !*noBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Println("open browser failed:", err)
		}
	}
	select {}
}
