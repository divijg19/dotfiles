package app

import (
	"encoding/json"
	"fmt"
	"time"
	"update-go-tools/internal/tool"
)

type JSONRenderer struct{}

func (JSONRenderer) Inventory(loadRes tool.LoadResult) error {
	toolsRep := make([]ToolReport, 0, len(loadRes.Tools))
	for _, t := range loadRes.Tools {
		toolsRep = append(toolsRep, ToolReport{
			Name:        t.Name(),
			Version:     t.Version(),
			PackagePath: t.PackagePath(),
			ModulePath:  t.ModulePath(),
		})
	}
	return emitJSON(ListReport{Tools: toolsRep})
}

func (JSONRenderer) Verify(loadRes tool.LoadResult) error {
	results := tool.Verify(loadRes.Tools)

	verReports := make([]VerifyResultReport, 0, len(results)+len(loadRes.Invalid))
	unhealthy := 0
	for _, r := range results {
		if !r.Healthy {
			unhealthy++
		}
		verReports = append(verReports, VerifyResultReport{
			Name:    r.Tool.Name(),
			Healthy: r.Healthy,
			Error:   r.Error,
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
	if err := emitJSON(VerifyReport{Results: verReports}); err != nil {
		return err
	}
	if unhealthy > 0 {
		return fmt.Errorf("%d unhealthy binaries found", unhealthy)
	}
	return nil
}

func (JSONRenderer) Outdated(outdatedRes []tool.OutdatedResult) error {
	outReports := make([]OutdatedItemReport, 0, len(outdatedRes))
	for _, o := range outdatedRes {
		errStr := ""
		if o.Error != nil {
			errStr = o.Error.Error()
		}
		outReports = append(outReports, OutdatedItemReport{
			Name:     o.Tool.Name(),
			Current:  o.Current,
			Latest:   o.Latest,
			Outdated: o.Outdated,
			Error:    errStr,
		})
	}
	return emitJSON(OutdatedReport{Results: outReports})
}

func (JSONRenderer) Update(results []tool.ToolUpdateResult, loadRes tool.LoadResult, duration time.Duration, diagnostics []tool.Diagnostic, checkOnly bool) error {
	skipped := make([]string, 0)
	for _, r := range results {
		if r.Status == tool.StatusSkippedLocal {
			skipped = append(skipped, r.Tool.Name())
		}
	}
	for _, inv := range loadRes.Invalid {
		skipped = append(skipped, inv.Path)
	}

	updatedList := make([]string, 0, len(results))
	notesList := make([]string, 0)
	failedList := make([]string, 0)

	for _, res := range results {
		switch res.Status {
		case tool.StatusSkippedLocal:
			continue
		case tool.StatusFailed:
			failedList = append(failedList, res.Tool.Name())
		default:
			updatedList = append(updatedList, res.Tool.Name())
			if len(res.Notes) > 0 {
				notesList = append(notesList, res.Tool.Name())
			}
		}
	}

	report := UpdateReport{
		Updated:   updatedList,
		Notes:     notesList,
		Skipped:   skipped,
		Failed:    failedList,
		CheckOnly: checkOnly,
	}
	if err := emitJSON(report); err != nil {
		return err
	}

	if len(failedList) > 0 {
		return fmt.Errorf("%d updates failed", len(failedList))
	}
	return nil
}

func (JSONRenderer) Info(loadRes tool.LoadResult, target string) error {
	for _, t := range loadRes.Tools {
		if t.Name() == target {
			return emitJSON(ToolReport{
				Name:        t.Name(),
				Version:     t.Version(),
				PackagePath: t.PackagePath(),
				ModulePath:  t.ModulePath(),
			})
		}
	}
	return fmt.Errorf("tool '%s' not found or has no module metadata", target)
}

func emitJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode JSON output: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
