package app

import (
	"encoding/json"
	"fmt"
	"update-go-tools/internal/tool"
)

type JSONRenderer struct{}

func (JSONRenderer) Inventory(report InventoryReport) error {
	toolsRep := make([]ToolReport, 0, len(report.Tools))
	for _, t := range report.Tools {
		toolsRep = append(toolsRep, ToolReport{
			Name:        t.Name,
			Version:     t.Version,
			PackagePath: t.PackagePath,
			ModulePath:  t.ModulePath,
		})
	}
	return emitJSON(ListReport{Tools: toolsRep})
}

func (JSONRenderer) Verify(report VerifyReport) error {
	if err := emitJSON(report); err != nil {
		return err
	}
	if report.Summary.Unhealthy > 0 {
		return fmt.Errorf("%d unhealthy binaries found", report.Summary.Unhealthy)
	}
	return nil
}

func (JSONRenderer) Outdated(report OutdatedReport) error {
	outReports := make([]OutdatedItemReport, 0, len(report.Results))
	for _, o := range report.Results {
		outReports = append(outReports, OutdatedItemReport{
			Name:     o.Name,
			Current:  o.Current,
			Latest:   o.Latest,
			Outdated: o.Outdated,
			Error:    o.Error,
		})
	}
	return emitJSON(OutdatedReport{Results: outReports})
}

func (JSONRenderer) Update(report UpdateReport) error {
	return emitJSON(report)
}

func (JSONRenderer) Check(report CheckReport) error {
	updated := make([]string, 0, len(report.CheckTargets))
	for _, ct := range report.CheckTargets {
		updated = append(updated, ct.Name)
	}
	return emitJSON(UpdateReport{
		Updated:   updated,
		Skipped:   []string{},
		Failed:    []string{},
		CheckOnly: true,
	})
}

func (JSONRenderer) DryRun(report DryRunReport) error {
	return emitJSON(report)
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