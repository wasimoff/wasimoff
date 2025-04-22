package main

import (
	"fmt"
	"log"
	"runtime/debug"
)

// Go toolchain version, main module path and Git information from build metadata.
type VersionInfo struct {
	Go     string
	Path   string
	Commit string
}

type vcsInfo struct {
	revision string
	modified string
}

var version VersionInfo = VersionInfo{}

func printVersion() {
	fmt.Printf("   %s (%s) %s\n", version.Path, version.Commit, version.Go)
	fmt.Println()
}

func init() {

	// read build information from binary
	// https://pkg.go.dev/runtime/debug#ReadBuildInfo
	info, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatalf("failed reading build information from binary")
	}

	// get basic go build information
	version.Go = info.GoVersion
	version.Path = info.Path

	// try to get the git revision from buildsettings
	// https://pkg.go.dev/runtime/debug#BuildSetting
	version.Commit = "devel"
	vcs := vcsInfo{}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			vcs.revision = setting.Value
		}
		if setting.Key == "vcs.modified" {
			vcs.modified = "-dirty"
		}
	}
	if vcs.revision != "" {
		version.Commit = fmt.Sprintf("%.*s%s", 7, vcs.revision, vcs.modified)
	}

}
