package info

import (
	"runtime/debug"
	"time"

	"github.com/rs/zerolog/log"
)

type Data struct {
	GoVersion string
	GoArch    string
	GoOS      string
	GoC       bool
	GitCommit string
	GitDate   time.Time
	GitDirty  bool
}

func NewInfo() *Data {
	info, _ := debug.ReadBuildInfo()
	data := &Data{
		GoVersion: info.GoVersion,
	}

	// Available info: https://github.com/golang/go/blob/master/src/runtime/debug/mod.go#L73
	for _, kv := range info.Settings {
		applyBuildSetting(data, kv.Key, kv.Value)
	}

	return data
}

func applyBuildSetting(data *Data, key, value string) {
	switch key {
	case "GOARCH":
		data.GoArch = value
	case "GOOS":
		data.GoOS = value
	case "CGO_ENABLED":
		data.GoC = value == "1"
	case "vcs.revision":
		data.GitCommit = value
	case "vcs.time":
		data.GitDate, _ = time.Parse(time.RFC3339, value)
	case "vcs.modified":
		data.GitDirty = value == "true"
	}
}

func (d *Data) Print() {
	log.Info().Msgf(`
Charttool is a tool for managing TrueCharts charts.

Go
    Version: %s
    OS: %s
    Arch: %s
    CGO: %t
Git
    Commit: %s
    Date: %s
    Dirty: %t
`, d.GoVersion, d.GoOS, d.GoArch, d.GoC, d.GitCommit, d.GitDate, d.GitDirty)
}
