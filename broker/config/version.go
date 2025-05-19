package config

import (
	"fmt"
	"log"
	"runtime/debug"
)

// Go toolchain version, main module path and Git information from build metadata.
type VersionInfo struct {
	GoVersion string `json:"go"`
	Package   string `json:"package"`
	Revision  string `json:"revision"`
}

var Version VersionInfo = VersionInfo{}

func init() {
	// read build information from binary
	// https://pkg.go.dev/runtime/debug#ReadBuildInfo
	info, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatalf("failed reading build information from binary")
	}

	// get basic go build information
	Version.GoVersion = info.GoVersion
	Version.Package = info.Path
	Version.Revision = "devel"

	// try to get the git revision from buildsettings
	// https://pkg.go.dev/runtime/debug#BuildSetting
	revision := ""
	dirtymarker := ""
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value
		}
		if setting.Key == "vcs.modified" && setting.Value == "true" {
			dirtymarker = "-dirty"
		}
	}
	if revision != "" {
		Version.Revision = fmt.Sprintf("%.*s%s", 7, revision, dirtymarker)
	}

}
