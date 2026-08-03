package tool

import (
	"debug/buildinfo"
)

type Tool struct {
	name string
	path string
	info *buildinfo.BuildInfo
}

func (t Tool) Name() string {
	return t.name
}

func (t Tool) Path() string {
	return t.path
}

func (t Tool) PackagePath() string {
	if t.info == nil {
		return ""
	}
	return t.info.Path
}

func (t Tool) ModulePath() string {
	if t.info == nil {
		return ""
	}
	if t.info.Main.Path != "" {
		return t.info.Main.Path
	}
	return t.info.Path
}

func (t Tool) Version() string {
	if t.info == nil || t.info.Main.Version == "" {
		return "unknown"
	}
	return t.info.Main.Version
}

func (t Tool) GoVersion() string {
	if t.info == nil || t.info.GoVersion == "" {
		return "unknown"
	}
	return t.info.GoVersion
}

func (t Tool) InstallTarget() string {
	return t.PackagePath()
}

func (t Tool) CanUpdate() bool {
	pkg := t.PackagePath()
	if pkg == "" || pkg == "(devel)" {
		return false
	}
	ver := t.Version()
	if ver == "" || ver == "unknown" || ver == "(devel)" {
		return false
	}
	return true
}

func (t Tool) IsValid() bool {
	return t.info != nil && t.PackagePath() != ""
}
