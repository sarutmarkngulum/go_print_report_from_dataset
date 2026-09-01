package renderer

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"go_print_report_from_dataset/internal/domain/report"
)

type HTMLRenderer struct {
	template *template.Template
}

func NewHTMLRenderer() (*HTMLRenderer, error) {
	tmpl, err := template.New("preview").Funcs(template.FuncMap{"cell": Cell}).Parse(previewTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse preview template: %w", err)
	}
	return &HTMLRenderer{template: tmpl}, nil
}

func (r *HTMLRenderer) Render(w io.Writer, preview report.Preview) error {
	if err := r.template.Execute(w, preview); err != nil {
		return fmt.Errorf("render preview: %w", err)
	}
	return nil
}

func BuildPreview(dataset report.Dataset) report.Preview {
	rows := dataset.Items.ReportData
	translations := ResolveTranslations(dataset.Items.TranslateReports)
	headerBands, subColumns := ResolveTableHeader(dataset.Items.DataSetItem, translations)
	columns := ResolveColumns(rows, dataset.Items.DataSetItem, translations)
	return report.Preview{
		Headers:     ResolveHeaders(dataset.ReportHead, rows, dataset.Items.DataSetHead, translations),
		Columns:     columns,
		HeaderBands: headerBands,
		SubColumns:  subColumns,
		Rows:        rows,
		DebugLogs:   ResolveDebugLogs(dataset, columns),
	}
}

func ResolveDebugLogs(dataset report.Dataset, columns []report.Column) []report.DebugLog {
	logs := make([]report.DebugLog, 0)
	rows := dataset.Items.ReportData
	metas := dataset.Items.DataSetItem
	translations := ResolveTranslations(dataset.Items.TranslateReports)

	if len(metas) == 0 {
		if len(rows) > 0 {
			logs = append(logs, report.DebugLog{Level: "info", Kind: "schema", Message: "ไม่พบ s_report_data_set_item จึงใช้ key จาก row แรกเป็น column สำรอง"})
		}
	}

	seenMeta := make(map[string]int)
	declared := make(map[string]bool)
	for _, meta := range metas {
		field := strings.TrimSpace(meta.Value)
		if field == "" {
			if strings.TrimSpace(meta.Caption) == "" {
				logs = append(logs, report.DebugLog{Level: "warning", Kind: "caption", Message: "group column ไม่มี caption"})
			}
			continue
		}
		seenMeta[field]++
		declared[field] = true
		translated := ""
		if meta.Tag != "" {
			translated = translations[meta.Tag]
		}
		if strings.TrimSpace(meta.Caption) == "" && strings.TrimSpace(translated) == "" {
			logs = append(logs, report.DebugLog{Level: "warning", Kind: "caption", Field: field, Message: "column ไม่มี caption และไม่มี translation"})
		}
	}
	for field, count := range seenMeta {
		if count > 1 {
			logs = append(logs, report.DebugLog{Level: "warning", Kind: "duplicate", Field: field, Message: fmt.Sprintf("metadata ระบุ field ซ้ำ %d ครั้ง", count)})
		}
	}
	for _, meta := range dataset.Items.DataSetHead {
		field := strings.TrimSpace(meta.Value)
		if field == "" {
			continue
		}
		_, inHead := dataset.ReportHead[field]
		_, inFirstRow := firstRowValue(rows, field)
		if !inHead && !inFirstRow {
			logs = append(logs, report.DebugLog{Level: "error", Kind: "head map", Field: field, Message: "หัวรายงานหา field นี้ไม่เจอทั้งใน report_head และ row แรก"})
		}
	}

	if len(rows) == 0 {
		logs = append(logs, report.DebugLog{Level: "warning", Kind: "data", Message: "ไม่พบ row ใน s_report_data"})
	}

	for field := range declared {
		missingRows := make([]string, 0)
		for index, row := range rows {
			if _, ok := row[field]; !ok {
				missingRows = append(missingRows, strconv.Itoa(index+1))
			}
		}
		if len(missingRows) > 0 {
			logs = append(logs, report.DebugLog{
				Level: "error", Kind: "map", Field: field, Rows: strings.Join(missingRows, ", "),
				Message: "หา field นี้ใน row ไม่เจอ จึง map ค่าไม่ได้",
			})
		}
	}

	if len(metas) > 0 && len(rows) > 0 {
		for field := range rows[0] {
			if !declared[field] {
				logs = append(logs, report.DebugLog{Level: "info", Kind: "extra", Field: field, Message: "พบ key ใน row แต่ไม่มี metadata column จึงไม่แสดงในตาราง"})
			}
		}
	}

	for _, column := range columns {
		if column.Datatype != "N" {
			continue
		}
		badRows := make([]string, 0)
		for index, row := range rows {
			raw := stringify(row[column.Key])
			if raw != "" {
				if _, ok := parseNumber(raw); !ok {
					badRows = append(badRows, strconv.Itoa(index+1))
				}
			}
		}
		if len(badRows) > 0 {
			logs = append(logs, report.DebugLog{Level: "warning", Kind: "value", Field: column.Key, Rows: strings.Join(badRows, ", "), Message: "ค่าไม่ใช่ตัวเลขตาม datatype=N จึงแสดง raw value เดิม"})
		}
	}

	sort.SliceStable(logs, func(i, j int) bool {
		if logs[i].Field != logs[j].Field {
			return logs[i].Field < logs[j].Field
		}
		return logs[i].Kind < logs[j].Kind
	})
	return logs
}

func ResolveTranslations(metas []report.TranslateMeta) map[string]string {
	if len(metas) == 0 {
		return nil
	}

	bySeq := make(map[string]map[string]string, len(metas))
	for _, meta := range metas {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(meta.Report), &parsed); err != nil {
			continue
		}
		seq := strings.ToLower(strings.TrimSpace(meta.Seq))
		if bySeq[seq] == nil {
			bySeq[seq] = make(map[string]string, len(parsed))
		}
		for key, value := range parsed {
			bySeq[seq][key] = value
		}
	}

	resolved := make(map[string]string)
	for key, value := range bySeq["default"] {
		resolved[key] = value
	}
	for key, value := range bySeq["main"] {
		resolved[key] = value
	}
	if len(resolved) == 0 {
		return nil
	}
	return resolved
}

func ResolveHeaders(reportHead map[string]any, rows []report.Row, metas []report.ColumnMeta, translations ...map[string]string) []report.HeaderField {
	if len(metas) == 0 {
		return nil
	}

	sortColumnMeta(metas)
	headers := make([]report.HeaderField, 0, len(metas))
	for _, meta := range metas {
		if meta.Value == "" {
			continue
		}
		headers = append(headers, report.HeaderField{
			Caption: captionFor(meta, translations...),
			Value:   lookupValue(reportHead, rows, meta.Value),
		})
	}
	return headers
}

func ResolveColumns(rows []report.Row, metas []report.ColumnMeta, translations ...map[string]string) []report.Column {
	if len(metas) > 0 {
		sortColumnMeta(metas)
		columns := make([]report.Column, 0, len(metas))
		for _, meta := range metas {
			if meta.Value == "" {
				continue
			}
			columns = append(columns, columnFromMeta(meta, translations...))
		}
		if len(columns) > 0 {
			return columns
		}
	}

	if len(rows) == 0 {
		return nil
	}
	keys := make([]string, 0, len(rows[0]))
	for key := range rows[0] {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	columns := make([]report.Column, 0, len(keys))
	for _, key := range keys {
		columns = append(columns, report.Column{Key: key, Caption: key, Datatype: inferDatatype(key)})
	}
	return columns
}

func ResolveTableHeader(metas []report.ColumnMeta, translations ...map[string]string) ([]report.HeaderBand, []report.Column) {
	if len(metas) == 0 {
		return nil, nil
	}

	sortColumnMeta(metas)
	hasGroup := false
	for _, meta := range metas {
		if meta.Value == "" {
			hasGroup = true
			break
		}
	}
	if !hasGroup {
		return nil, nil
	}

	bands := make([]report.HeaderBand, 0, len(metas))
	subColumns := make([]report.Column, 0, len(metas))
	for i := 0; i < len(metas); i++ {
		meta := metas[i]
		if meta.Value != "" {
			bands = append(bands, report.HeaderBand{Caption: captionFor(meta, translations...), ColSpan: 1, RowSpan: 2})
			continue
		}

		span := 0
		for j := i + 1; j < len(metas) && metas[j].Value != ""; j++ {
			span++
		}
		if span > 0 {
			bands = append(bands, report.HeaderBand{Caption: firstNonEmpty(meta.Caption, meta.Value), ColSpan: span})
			for j := i + 1; j <= i+span; j++ {
				subColumns = append(subColumns, columnFromMeta(metas[j], translations...))
			}
			i += span
		}
	}
	return bands, subColumns
}

func Cell(row report.Row, column any) string {
	if row == nil {
		return ""
	}
	switch typed := column.(type) {
	case report.Column:
		return formatValue(row[typed.Key], typed)
	case string:
		return stringify(row[typed])
	default:
		return ""
	}
}

func columnFromMeta(meta report.ColumnMeta, translations ...map[string]string) report.Column {
	return report.Column{
		Key:      meta.Value,
		Caption:  captionFor(meta, translations...),
		Datatype: meta.Datatype,
		Digit:    meta.Digit,
		Format:   meta.Format,
		Width:    meta.Width,
	}
}

func captionFor(meta report.ColumnMeta, translations ...map[string]string) string {
	if meta.Tag != "" {
		for _, translation := range translations {
			if value := translation[meta.Tag]; value != "" {
				return value
			}
		}
	}
	return firstNonEmpty(meta.Caption, meta.Value)
}

func formatValue(value any, column report.Column) string {
	raw := stringify(value)
	if column.Datatype != "N" || raw == "" {
		return raw
	}

	number, ok := parseNumber(raw)
	if !ok {
		return raw
	}

	digit, hasDigit := parseDigit(column.Digit)
	formatted := raw
	if hasDigit {
		formatted = strconv.FormatFloat(math.Abs(number), 'f', digit, 64)
	} else if column.Format == "-=()" && number < 0 {
		formatted = strings.TrimPrefix(raw, "-")
	}

	if column.Format == "-=()" && number < 0 {
		return "(" + formatted + ")"
	}
	if hasDigit {
		return strconv.FormatFloat(number, 'f', digit, 64)
	}
	return raw
}

func parseNumber(value string) (float64, bool) {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	number, err := strconv.ParseFloat(normalized, 64)
	return number, err == nil
}

func parseDigit(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	digit, err := strconv.Atoi(value)
	if err != nil || digit < 0 {
		return 0, false
	}
	return digit, true
}

func lookupValue(reportHead map[string]any, rows []report.Row, key string) string {
	if value := stringify(reportHead[key]); value != "" {
		return value
	}
	if len(rows) == 0 {
		return ""
	}
	return stringify(rows[0][key])
}

func firstRowValue(rows []report.Row, key string) (any, bool) {
	if len(rows) == 0 {
		return nil, false
	}
	value, ok := rows[0][key]
	return value, ok
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func sortColumnMeta(metas []report.ColumnMeta) {
	sort.SliceStable(metas, func(i, j int) bool {
		return columnIndex(metas[i]) < columnIndex(metas[j])
	})
}

func columnIndex(meta report.ColumnMeta) int {
	idx, err := strconv.Atoi(meta.Index)
	if err != nil {
		return 0
	}
	return idx
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func inferDatatype(key string) string {
	if len(key) >= 6 && (key[:6] == "n_col_" || key[:6] == "s_col_") {
		return "N"
	}
	return "C"
}

const previewTemplate = `<!doctype html>
<html lang="th">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Report Dataset Preview</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Tahoma,sans-serif;margin:24px;color:#1f2933;background:#f7f7f4}
.toolbar{display:flex;gap:8px;margin-bottom:16px}
button,a.button{border:1px solid #8a8f6a;background:#596235;color:white;padding:8px 12px;border-radius:4px;text-decoration:none;cursor:pointer;font:inherit}
.sheet{background:white;padding:20px;border:1px solid #d7d7cf;overflow:auto}
.report-head{display:grid;grid-template-columns:max-content 1fr;gap:4px 14px;margin-bottom:16px;font-size:14px}
.report-head dt{font-weight:600;color:#4b5563}
.report-head dd{margin:0}
table{border-collapse:collapse;width:100%;font-size:13px}
th,td{border:1px solid #cfd4dc;padding:6px 8px;vertical-align:top;white-space:nowrap}
th{background:#eef0e7;text-align:center;font-weight:600}
td.text{text-align:left}
td.number{text-align:right;font-variant-numeric:tabular-nums}
@media print{body{margin:0;background:white}.toolbar{display:none}.sheet{border:0;padding:0}th{background:#eee!important;-webkit-print-color-adjust:exact;print-color-adjust:exact}}
.debug{margin-top:20px;border-top:2px solid #d7d7cf;padding-top:12px}.debug h2{font-size:16px;margin:0 0 8px}.debug table{font-size:12px}.debug th{background:#f2f3f0}.debug .error{color:#a12622}.debug .warning{color:#8a5a00}.debug .info{color:#315c8a}
</style>
</head>
<body>
<div class="toolbar">
<button onclick="window.print()">Print</button>
<a class="button" href="/">Back</a>
</div>
<main class="sheet">
{{if .Headers}}
<dl class="report-head">
{{range .Headers}}<dt>{{.Caption}}</dt><dd>{{.Value}}</dd>{{end}}
</dl>
{{end}}
<table>
<colgroup>{{range .Columns}}<col{{if .Width}} style="width:{{.Width}}%" data-width="{{.Width}}"{{end}}>{{end}}</colgroup>
<thead>
{{if .HeaderBands}}
<tr>{{range .HeaderBands}}<th colspan="{{.ColSpan}}"{{if .RowSpan}} rowspan="{{.RowSpan}}"{{end}}>{{.Caption}}</th>{{end}}</tr>
<tr>{{range .SubColumns}}<th>{{.Caption}}</th>{{end}}</tr>
{{else}}
<tr>{{range .Columns}}<th>{{.Caption}}</th>{{end}}</tr>
{{end}}
</thead>
<tbody>
{{range .Rows}}
{{$row := .}}
<tr>{{range $.Columns}}<td class="{{.AlignClass}}">{{cell $row .}}</td>{{end}}</tr>
{{end}}
</tbody>
</table>
{{if .DebugLogs}}
<section class="debug">
<h2>Debug mapping</h2>
<table>
<thead><tr><th>ระดับ</th><th>ประเภท</th><th>Field</th><th>Row</th><th>รายละเอียด</th></tr></thead>
<tbody>{{range .DebugLogs}}<tr><td class="{{.Level}}">{{.Level}}</td><td>{{.Kind}}</td><td>{{.Field}}</td><td>{{.Rows}}</td><td>{{.Message}}</td></tr>{{end}}</tbody>
</table>
</section>
{{end}}
</main>
</body>
</html>`
