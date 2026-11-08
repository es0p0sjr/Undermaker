// ucdev serves only a validated local snapshot. It never synchronizes data.
package main

import (
	"fmt"
	"os"
	"ucprint/internal/app"
)

func main() {
	data := "data"
	args := os.Args[1:]
	if len(args) >= 2 && args[0] == "--data" {
		data = args[1]
		args = args[2:]
	}
	if e := app.Run(os.DirFS(data), os.DirFS("web"), true, args); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
