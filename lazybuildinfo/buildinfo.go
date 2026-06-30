package lazybuildinfo

import (
	"runtime/debug"
	"strings"
)

type buildInfo struct {
	Available bool           `json:"available"`
	GoVersion string         `json:"go_version,omitempty"`
	Path      string         `json:"path,omitempty"`
	Main      module         `json:"main"`
	Deps      []module       `json:"deps,omitempty"`
	Settings  []buildSetting `json:"settings,omitempty"`
}

type module struct {
	Path    string  `json:"path"`
	Version string  `json:"version,omitempty"`
	Sum     string  `json:"sum,omitempty"`
	Replace *module `json:"replace,omitempty"`
}

type buildSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func snapshot() buildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return buildInfo{}
	}
	out := buildInfo{
		Available: true,
		GoVersion: info.GoVersion,
		Path:      info.Path,
		Main:      moduleFromDebug(info.Main),
		Deps:      make([]module, 0, len(info.Deps)),
		Settings:  make([]buildSetting, 0, len(info.Settings)),
	}
	for _, dep := range info.Deps {
		if dep == nil {
			continue
		}
		out.Deps = append(out.Deps, moduleFromDebug(*dep))
	}
	for _, setting := range info.Settings {
		out.Settings = append(out.Settings, buildSetting{
			Key:   setting.Key,
			Value: setting.Value,
		})
	}
	return out
}

// Version returns the best available application build version.
//
// It prefers the main module version recorded by the Go tool, then the VCS
// revision setting, and falls back to "devel" when neither value is available.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}
	version := strings.TrimSpace(info.Main.Version)
	if version != "" && version != "(devel)" {
		return version
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && strings.TrimSpace(setting.Value) != "" {
			return strings.TrimSpace(setting.Value)
		}
	}
	return "devel"
}

func moduleFromDebug(in debug.Module) module {
	out := module{
		Path:    in.Path,
		Version: in.Version,
		Sum:     in.Sum,
	}
	if in.Replace != nil {
		replacement := moduleFromDebug(*in.Replace)
		out.Replace = &replacement
	}
	return out
}
