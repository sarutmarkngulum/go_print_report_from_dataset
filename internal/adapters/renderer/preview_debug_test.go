package renderer

import (
	"strings"
	"testing"

	"go_print_report_from_dataset/internal/domain/report"
)

func TestResolveDebugLogsReportsMappingProblems(t *testing.T) {
	dataset := report.Dataset{
		Items: report.Items{
			DataSetItem: []report.ColumnMeta{
				{Index: "1", Value: "amount", Datatype: "N", Caption: "Amount"},
				{Index: "2", Value: "amount", Datatype: "N", Caption: "Amount again"},
				{Index: "3", Value: "missing", Caption: "Missing"},
				{Index: "4", Value: "", Caption: ""},
			},
			ReportData: []report.Row{
				{"amount": "not-a-number", "unused": "x"},
				{"amount": 5},
			},
		},
	}

	preview := BuildPreview(dataset)
	if !hasDebug(preview.DebugLogs, "duplicate", "amount") {
		t.Fatal("expected duplicate metadata debug log")
	}
	if !hasDebug(preview.DebugLogs, "map", "missing") {
		t.Fatal("expected missing field debug log")
	}
	if !hasDebug(preview.DebugLogs, "extra", "unused") {
		t.Fatal("expected extra field debug log")
	}
	if !hasDebug(preview.DebugLogs, "value", "amount") {
		t.Fatal("expected invalid numeric value debug log")
	}
	if !hasDebug(preview.DebugLogs, "caption", "") {
		t.Fatal("expected empty group caption debug log")
	}
}

func TestRenderIncludesDebugPanel(t *testing.T) {
	renderer, err := NewHTMLRenderer()
	if err != nil {
		t.Fatal(err)
	}
	preview := report.Preview{DebugLogs: []report.DebugLog{{Level: "error", Kind: "map", Field: "missing", Rows: "1", Message: "หา field ไม่เจอ"}}}
	var output strings.Builder
	if err := renderer.Render(&output, preview); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Debug mapping") || !strings.Contains(output.String(), "missing") {
		t.Fatal("expected debug panel in rendered HTML")
	}
}

func hasDebug(logs []report.DebugLog, kind, field string) bool {
	for _, entry := range logs {
		if entry.Kind == kind && entry.Field == field {
			return true
		}
	}
	return false
}
