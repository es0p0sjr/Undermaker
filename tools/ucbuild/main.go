// ucbuild gates release generation on a valid, complete snapshot.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"ucprint/internal/snapshot"
)

func copyFile(root, name string, b []byte) error {
	p := filepath.Join(root, filepath.FromSlash(name))
	if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
		return e
	}
	return os.WriteFile(p, b, 0644)
}
func build(data, output string) error {
	d, e := snapshot.Load(os.DirFS(data), true)
	if e != nil {
		return fmt.Errorf("build blocked: %w", e)
	}
	npm := "npm"
	if runtime.GOOS == "windows" {
		npm = "npm.cmd"
	}
	cmd := exec.Command(npm, "run", "build")
	cmd.Env = append(os.Environ(), "UC_DATA="+data)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e = cmd.Run(); e != nil {
		return e
	}
	base, e := filepath.Abs(".build")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(base, 0755); e != nil {
		return e
	}
	stage, e := os.MkdirTemp(base, "release-")
	if e != nil {
		return e
	}
	defer func() {
		rel, _ := filepath.Rel(base, stage)
		if rel != "." && !strings.HasPrefix(rel, "..") {
			_ = os.RemoveAll(stage)
		}
	}()
	for name, b := range d.Files {
		if !strings.HasPrefix(name, "raw/") {
			if e = copyFile(stage, "data/"+name, b); e != nil {
				return e
			}
		}
	}
	e = fs.WalkDir(os.DirFS(".build/frontend"), ".", func(name string, entry fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if entry.IsDir() {
			return nil
		}
		b, e := os.ReadFile(filepath.Join(".build/frontend", filepath.FromSlash(name)))
		if e != nil {
			return e
		}
		return copyFile(stage, "web/"+name, b)
	})
	if e != nil {
		return e
	}
	if _, e = snapshot.Load(os.DirFS(filepath.Join(stage, "data")), false); e != nil {
		return fmt.Errorf("embedded staging failed validation: %w", e)
	}
	main := `package main
import("embed";"fmt";"io/fs";"os";"ucprint/internal/app")
//go:embed web data
var bundle embed.FS
func main(){data,_:=fs.Sub(bundle,"data");web,_:=fs.Sub(bundle,"web");if e:=app.Run(data,web,false,os.Args[1:]);e!=nil{fmt.Fprintln(os.Stderr,e);if len(os.Args)==1{fmt.Fprintln(os.Stderr,"Press Enter to close.");fmt.Scanln()};os.Exit(1)}}
`
	if e = copyFile(stage, "main.go", []byte(main)); e != nil {
		return e
	}
	temp := filepath.Join(stage, "uc-print.exe")
	cwd, _ := os.Getwd()
	relative, e := filepath.Rel(cwd, stage)
	if e != nil {
		return e
	}
	goCmd := exec.Command("go", "build", "-trimpath", "-o", temp, "./"+filepath.ToSlash(relative))
	goCmd.Env = append(os.Environ(), "GOPROXY=off", "GOTOOLCHAIN=local")
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr
	if e = goCmd.Run(); e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(output), 0755); e != nil {
		return e
	}
	if e = os.Rename(temp, output); e != nil {
		return fmt.Errorf("could not publish executable (close an existing running copy): %w", e)
	}
	fmt.Println("Built offline executable:", output)
	return nil
}
func main() {
	data := flag.String("data", "data", "validated snapshot directory")
	output := flag.String("output", "dist/uc-print.exe", "output executable")
	flag.Parse()
	if e := build(*data, *output); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
