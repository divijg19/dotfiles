package app

import (
	"context"
	"time"
	"update-go-tools/internal/tool"
)

type App struct {
	Gobin        string
	Renderer     Renderer
	Runner       tool.Runner
	loadRes      tool.LoadResult
	loadResValid bool
}

func NewApp(renderer Renderer, runner tool.Runner) (*App, error) {
	gobin, err := tool.GetGobin()
	if err != nil {
		return nil, err
	}
	if termRend, ok := renderer.(*TerminalRenderer); ok {
		termRend.Gobin = gobin
	}
	return &App{
		Gobin:    gobin,
		Renderer: renderer,
		Runner:   runner,
	}, nil
}

func (a *App) load() (tool.LoadResult, error) {
	if a.loadResValid {
		return a.loadRes, nil
	}
	res, err := tool.Load(a.Gobin)
	if err != nil {
		return tool.LoadResult{}, err
	}
	a.loadRes = res
	a.loadResValid = true
	return res, nil
}

func (a *App) RunInventory() error {
	loadRes, err := a.load()
	if err != nil {
		return err
	}
	report := a.inventoryReport(loadRes)
	return a.Renderer.Inventory(report)
}

func (a *App) inventoryReport(loadRes tool.LoadResult) InventoryReport {
	tools := make([]ToolReport, 0, len(loadRes.Tools))
	for _, t := range loadRes.Tools {
		tools = append(tools, ToolReport{
			Name:        t.Name(),
			Version:     t.Version(),
			PackagePath: t.PackagePath(),
			ModulePath:  t.ModulePath(),
		})
	}
	invalid := make([]InvalidReport, 0, len(loadRes.Invalid))
	for _, inv := range loadRes.Invalid {
		invalid = append(invalid, InvalidReport{
			Path:    inv.Path,
			Message: inv.Message(),
		})
	}
	return InventoryReport{
		Tools:   tools,
		Invalid: invalid,
		Summary: loadRes.Summary,
	}
}

func (a *App) RunVerify() error {
	loadRes, err := a.load()
	if err != nil {
		return err
	}
	report := a.verifyReport(loadRes)
	return a.Renderer.Verify(report)
}

func (a *App) verifyReport(loadRes tool.LoadResult) VerifyReport {
	results := tool.Verify(loadRes.Tools)

	verReports := make([]VerifyResultReport, 0, len(results)+len(loadRes.Invalid))
	healthy := 0
	localCount := 0
	unhealthy := 0

	for _, r := range results {
		if r.Healthy {
			healthy++
			if !r.Tool.CanUpdate() {
				localCount++
			}
		} else {
			unhealthy++
		}
		verReports = append(verReports, VerifyResultReport{
			Name:        r.Tool.Name(),
			Version:     r.Tool.Version(),
			PackagePath: r.Tool.PackagePath(),
			Healthy:     r.Healthy,
			Error:       r.Error,
		})
	}
	for _, inv := range loadRes.Invalid {
		unhealthy++
		verReports = append(verReports, VerifyResultReport{
			Name:    inv.Path,
			Healthy: false,
			Error:   inv.Message(),
		})
	}

	return VerifyReport{
		Results: verReports,
		Summary: VerifySummary{
			Healthy:   healthy,
			Local:     localCount,
			Invalid:   len(loadRes.Invalid),
			Unhealthy: unhealthy,
		},
	}
}

func (a *App) RunOutdated(ctx context.Context) error {
	loadRes, err := a.load()
	if err != nil {
		return err
	}
	outdatedRes := tool.CheckOutdated(ctx, loadRes.Tools, a.Runner)
	report := a.outdatedReport(outdatedRes)
	return a.Renderer.Outdated(report)
}

func (a *App) outdatedReport(outdatedRes []tool.OutdatedResult) OutdatedReport {
	outReports := make([]OutdatedItemReport, 0, len(outdatedRes))
	outdatedCount := 0
	upToDateCount := 0

	for _, o := range outdatedRes {
		errStr := ""
		if o.Error != nil {
			errStr = o.Error.Error()
		}
		if o.Outdated {
			outdatedCount++
		} else {
			upToDateCount++
		}
		outReports = append(outReports, OutdatedItemReport{
			Name:     o.Tool.Name(),
			Current:  o.Current,
			Latest:   o.Latest,
			Outdated: o.Outdated,
			Error:    errStr,
		})
	}

	return OutdatedReport{
		Results: outReports,
		Summary: OutdatedSummary{
			Outdated: outdatedCount,
			UpToDate: upToDateCount,
		},
	}
}

func (a *App) RunInfo(target string) error {
	loadRes, err := a.load()
	if err != nil {
		return err
	}
	return a.Renderer.Info(loadRes, target)
}

// LoadTools returns the inventory of installed tools in a single pass.
func (a *App) LoadTools() (tool.LoadResult, error) {
	return a.load()
}

func (a *App) RunUpdate(ctx context.Context, args []string, checkOnly bool) error {
	loadRes, err := a.load()
	if err != nil {
		return err
	}

	var onProgress func(tool.Progress)
	if termRend, ok := a.Renderer.(interface{ OnProgress(tool.Progress) }); ok {
		onProgress = termRend.OnProgress
	}

	var updatableTools []tool.Tool
	for _, t := range loadRes.Tools {
		if t.CanUpdate() {
			updatableTools = append(updatableTools, t)
		}
	}

	results, duration, diagnostics := tool.Update(ctx, updatableTools, args, checkOnly, a.Runner, onProgress)
	report := a.updateReport(results, loadRes, duration, diagnostics, checkOnly)
	return a.Renderer.Update(report)
}

func (a *App) updateReport(results []tool.ToolUpdateResult, loadRes tool.LoadResult, duration time.Duration, diagnostics []tool.Diagnostic, checkOnly bool) UpdateReport {
	updated := make([]string, 0)
	notes := make([]string, 0)
	failed := make([]string, 0)
	checkTargets := make([]CheckTarget, 0)

	for _, res := range results {
		if res.Status == tool.StatusSkippedLocal {
			continue
		}
		if res.Success {
			updated = append(updated, res.Tool.Name())
			if len(res.Notes) > 0 {
				notes = append(notes, res.Tool.Name())
			}
		} else {
			failed = append(failed, res.Tool.Name())
		}
		if checkOnly {
			checkTargets = append(checkTargets, CheckTarget{
				Name:          res.Tool.Name(),
				InstallTarget: res.Tool.InstallTarget(),
			})
		}
	}

	skipped := make([]string, 0, len(loadRes.Invalid))
	for _, inv := range loadRes.Invalid {
		skipped = append(skipped, inv.Path)
	}

	return UpdateReport{
		Updated:      updated,
		Notes:        notes,
		Skipped:      skipped,
		Failed:       failed,
		CheckOnly:    checkOnly,
		Duration:     duration,
		Diagnostics:  diagnostics,
		CheckTargets: checkTargets,
	}
}

func (a *App) RunDryRun(ctx context.Context, args []string) error {
	loadRes, err := a.load()
	if err != nil {
		return err
	}

	plan := planUpdate(loadRes, args)
	report := a.dryRunReport(plan)
	return a.Renderer.DryRun(report)
}

func planUpdate(loadRes tool.LoadResult, filter []string) planResult {
	filterMap := make(map[string]bool)
	for _, f := range filter {
		filterMap[f] = true
	}

	var toUpdate []tool.Tool
	var skipped []tool.Tool

	for _, t := range loadRes.Tools {
		if len(filterMap) > 0 && !filterMap[t.Name()] {
			continue
		}
		if t.CanUpdate() {
			toUpdate = append(toUpdate, t)
		} else {
			skipped = append(skipped, t)
		}
	}

	return planResult{
		ToUpdate: toUpdate,
		Skipped:  skipped,
		Invalid:  loadRes.Invalid,
	}
}

type planResult struct {
	ToUpdate []tool.Tool
	Skipped  []tool.Tool
	Invalid  []tool.InvalidBinary
}

func (a *App) dryRunReport(plan planResult) DryRunReport {
	toUpdate := make([]DryRunItem, 0, len(plan.ToUpdate))
	for _, t := range plan.ToUpdate {
		toUpdate = append(toUpdate, DryRunItem{
			Name:          t.Name(),
			PackagePath:   t.PackagePath(),
			InstallTarget: t.InstallTarget(),
		})
	}

	skipped := make([]DryRunItem, 0, len(plan.Skipped)+len(plan.Invalid))
	for _, t := range plan.Skipped {
		skipped = append(skipped, DryRunItem{
			Name: t.Name(),
		})
	}
	for _, inv := range plan.Invalid {
		skipped = append(skipped, DryRunItem{
			Name: inv.Path,
		})
	}

	return DryRunReport{
		ToUpdate: toUpdate,
		Skipped:  skipped,
	}
}
