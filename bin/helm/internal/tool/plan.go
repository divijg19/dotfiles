package tool

// PlanResult is the immutable outcome of planning an update. It is produced by
// the domain layer so planning has exactly one owner; renderers and the app
// layer never re-derive which tools would update.
type PlanResult struct {
	ToUpdate []Tool
	Skipped  []Tool
	Invalid  []InvalidBinary
}

// Plan computes which tools would be updated for the given filter. An empty
// filter selects every tool; otherwise only tools whose name matches are
// considered. Local/devel tools and invalid binaries are reported separately
// so renderers never count them as update candidates. Invalid binaries are
// always reported (they cannot be matched by name).
func Plan(loadRes LoadResult, filter []string) PlanResult {
	set := nameSet(filter)

	var result PlanResult
	for _, t := range loadRes.Tools {
		if !selected(t.Name(), set) {
			continue
		}
		if t.CanUpdate() {
			result.ToUpdate = append(result.ToUpdate, t)
		} else {
			result.Skipped = append(result.Skipped, t)
		}
	}
	result.Invalid = append(result.Invalid, loadRes.Invalid...)
	return result
}

// InstallRef returns the module@version reference `go install` uses to update
// a tool. It is the single source of truth for what "updating" means, shared
// by the execution path and by the human-readable plan command.
func InstallRef(target string) string {
	return target + "@latest"
}

// InstallCommand returns the human-readable `go install` command for a tool.
// The plan renderer uses it; it always agrees with what InstallRef would run.
func InstallCommand(target string) string {
	return "go install " + InstallRef(target)
}

// nameSet builds a lookup set of selected tool names. An empty set means every
// tool is selected.
func nameSet(names []string) map[string]bool {
	if len(names) == 0 {
		return nil
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
}

// selected reports whether a tool name passes the given filter set.
func selected(name string, set map[string]bool) bool {
	return len(set) == 0 || set[name]
}
