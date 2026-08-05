package app

import (
	"context"
	"time"

	"helm/internal/tool"
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
	verifyResults := tool.Verify(loadRes.Tools)
	verifyMap := make(map[string]tool.VerificationResult)
	for _, vr := range verifyResults {
		verifyMap[vr.Tool.Name()] = vr
	}

	var items []ToolInventoryItem
	healthy := 0
	localCount := 0
	unhealthy := 0

	for _, t := range loadRes.Tools {
		vr, ok := verifyMap[t.Name()]
		status := "Healthy"
		errStr := ""
		if ok && !vr.Healthy {
			status = "Unhealthy"
			errStr = vr.Error
			unhealthy++
		} else if !t.CanUpdate() {
			status = "Local"
			localCount++
			healthy++
		} else {
			healthy++
		}

		items = append(items, ToolInventoryItem{
			Name:        t.Name(),
			Version:     t.Version(),
			PackagePath: t.PackagePath(),
			ModulePath:  t.ModulePath(),
			Status:      status,
			Error:       errStr,
		})
	}

	for range loadRes.Invalid {
		unhealthy++
	}

	return InventoryReport{
		OperationEnvelope: OperationEnvelope{
			Operation: OperationList,
			Success:   unhealthy == 0 && len(loadRes.Invalid) == 0,
		},
		Tools:   items,
		Invalid: a.invalidReports(loadRes.Invalid),
		Summary: InventorySummary{
			Healthy:   healthy,
			Local:     localCount,
			Invalid:   len(loadRes.Invalid),
			Unhealthy: unhealthy,
		},
	}
}

func (a *App) invalidReports(invalids []tool.InvalidBinary) []InvalidReport {
	var reps []InvalidReport
	for _, inv := range invalids {
		reps = append(reps, InvalidReport{
			Path:    inv.Path,
			Message: inv.Message(),
		})
	}
	return reps
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
		OperationEnvelope: OperationEnvelope{
			Operation: OperationOutdated,
			Success:   true,
		},
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

func (a *App) LoadTools() (tool.LoadResult, error) {
	return a.load()
}

func (a *App) RunPlan(ctx context.Context, args []string) error {
	loadRes, err := a.load()
	if err != nil {
		return err
	}
	plan := tool.Plan(loadRes, args)
	report := a.planReport(plan)
	return a.Renderer.Plan(report)
}

func (a *App) planReport(plan tool.PlanResult) PlanReport {
	toUpdate := make([]PlanItem, 0, len(plan.ToUpdate))
	for _, t := range plan.ToUpdate {
		toUpdate = append(toUpdate, PlanItem{
			Name:          t.Name(),
			PackagePath:   t.PackagePath(),
			InstallTarget: t.InstallTarget(),
			Command:       tool.InstallCommand(t.InstallTarget()),
		})
	}

	skipped := make([]PlanItem, 0, len(plan.Skipped)+len(plan.Invalid))
	for _, t := range plan.Skipped {
		skipped = append(skipped, PlanItem{Name: t.Name()})
	}
	for _, inv := range plan.Invalid {
		skipped = append(skipped, PlanItem{Name: inv.Path})
	}

	return PlanReport{
		OperationEnvelope: OperationEnvelope{
			Operation: OperationCheck,
			Success:   true,
		},
		WouldUpdate: toUpdate,
		Skipped:     skipped,
	}
}

func (a *App) RunUpdate(ctx context.Context, args []string) error {
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

	results, duration, diagnostics := tool.Update(ctx, updatableTools, args, false, a.Runner, onProgress)
	report := a.updateReport(results, loadRes, duration, diagnostics)
	return a.Renderer.Update(report)
}

func (a *App) updateReport(results []tool.ToolUpdateResult, loadRes tool.LoadResult, duration time.Duration, diagnostics []tool.Diagnostic) UpdateReport {
	updated := make([]string, 0)
	notes := make([]string, 0)
	failed := make([]string, 0)

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
	}

	skipped := make([]string, 0, len(loadRes.Invalid))
	for _, inv := range loadRes.Invalid {
		skipped = append(skipped, inv.Path)
	}

	return UpdateReport{
		OperationEnvelope: OperationEnvelope{
			Operation: OperationUpdate,
			Success:   len(failed) == 0,
		},
		Updated:     updated,
		Notes:       notes,
		Skipped:     skipped,
		Failed:      failed,
		Duration:    duration,
		Diagnostics: diagnostics,
	}
}
