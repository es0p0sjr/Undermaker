package app

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"syscall"
	"time"
	"ucprint/internal/snapshot"
)

func Run(dataFS, webFS fs.FS, raw bool, args []string) error {
	flags := flag.NewFlagSet("uc-print", flag.ContinueOnError)
	port := flags.Int("port", 8765, "local port")
	open := flags.Bool("open", len(args) == 0, "open local app in browser (default when launched without arguments)")
	if e := flags.Parse(args); e != nil {
		return e
	}
	data, e := snapshot.Load(dataFS, raw)
	if e != nil {
		if !raw {
			return fmt.Errorf("the embedded snapshot is damaged or incomplete; rebuild this executable from a validated snapshot: %w", e)
		}
		if _, manifestErr := fs.Stat(dataFS, "manifest.json"); errors.Is(manifestErr, fs.ErrNotExist) {
			return fmt.Errorf("UC Print setup is not complete.\n\nThe snapshot manifest is created automatically only after every card, rule and image has been imported successfully. Do not create manifest.json yourself.\n\nYou can run the browser artwork helper without starting UC Print. Follow tools/ucsync/BROWSER.md first.\n\nAfter the helper finishes, run from the project folder:\n  go run ./tools/ucsync import ./JSON/AllCards.Json --translations ./JSON/en.json --assets ./JSON/assets\n\nThen run start.bat again. The validate command checks a snapshot; it does not create one")
		}
		return fmt.Errorf("UC Print cannot start because the local snapshot failed validation: %w.\nRun go run ./tools/ucsync validate for details, then repair it with ucsync import. No files were downloaded or changed", e)
	}
	portSpecified := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			portSpecified = true
		}
	})
	listener, e := listenLocal(*port, !portSpecified)
	if e != nil {
		return e
	}
	url := "http://" + listener.Addr().String()
	fmt.Println("UC Print offline:", url)
	if *open {
		go func() {
			time.Sleep(250 * time.Millisecond)
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "windows":
				cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
			case "darwin":
				cmd = exec.Command("open", url)
			default:
				cmd = exec.Command("xdg-open", url)
			}
			_ = cmd.Run()
		}()
	}
	handler := New(data, webFS)
	handler.Reworks = reworksDirectory()
	server := http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	return server.Serve(listener)
}

// A second copy must serve its own embedded renderer, even when an older
// version is still using the default port. Explicit port requests stay strict.
func listenLocal(port int, allowFallback bool) (net.Listener, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	addressInUse := syscall.EADDRINUSE
	if runtime.GOOS == "windows" {
		addressInUse = syscall.Errno(10048) // Winsock WSAEADDRINUSE.
	}
	if errors.Is(err, addressInUse) {
		if allowFallback {
			return net.Listen("tcp", "127.0.0.1:0")
		}
		return nil, fmt.Errorf("local port %d is already in use; close the other app or use --port 0 to choose an available port: %w", port, err)
	}
	return listener, err
}
