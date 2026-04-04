package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/0to1a/sweo/internal/cli"
)

// version is set by -ldflags at build time (via Makefile).
// Falls back to Go module version (set by go install).
var version = ""

func main() {
	if version == "" {
		version = moduleVersion()
	}
	root := cli.NewRootCmd(version)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func moduleVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
