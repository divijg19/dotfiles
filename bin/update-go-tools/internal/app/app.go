package app

import (
	"context"
	"update-go-tools/internal/tool"
)

type App struct {
	Gobin    string
	Renderer Renderer
	Runner   tool.Runner
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

func (a *App) RunInventory() error {
	loadRes, err := tool.Load(a.Gobin)
	if err != nil {
		return err
	}
	return a.Renderer.Inventory(loadRes)
}

func (a *App) RunVerify() error {
	loadRes, err := tool.Load(a.Gobin)
	if err != nil {
		return err
	}
	return a.Renderer.Verify(loadRes)
}

func (a *App) RunOutdated(ctx context.Context) error {
	loadRes, err := tool.Load(a.Gobin)
	if err != nil {
		return err
	}
	outdatedRes := tool.CheckOutdated(ctx, loadRes.Tools, a.Runner)
	return a.Renderer.Outdated(outdatedRes)
}

func (a *App) RunInfo(target string) error {
	loadRes, err := tool.Load(a.Gobin)
	if err != nil {
		return err
	}
	return a.Renderer.Info(loadRes, target)
}

// LoadTools returns the inventory of installed tools in a single pass.
func (a *App) LoadTools() (tool.LoadResult, error) {
	return tool.Load(a.Gobin)
}

func (a *App) RunUpdate(ctx context.Context, args []string, checkOnly bool, loadRes tool.LoadResult) error {
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
	return a.Renderer.Update(results, loadRes, duration, diagnostics, checkOnly)
}
