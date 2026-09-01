package report

type Dataset struct {
	ReportHead map[string]any `json:"report_head"`
	Items      Items          `json:"items"`
}

type Items struct {
	ReportData       []Row           `json:"s_report_data"`
	DataSetHead      []ColumnMeta    `json:"s_report_data_set_head"`
	DataSetItem      []ColumnMeta    `json:"s_report_data_set_item"`
	TranslateReports []TranslateMeta `json:"s_report_data_set_translate_report"`
}

type Row map[string]any

type ColumnMeta struct {
	Caption  string `json:"s_item_column_caption"`
	Datatype string `json:"s_item_column_datatype"`
	Digit    string `json:"s_item_column_digit"`
	Format   string `json:"s_item_column_format"`
	Index    string `json:"s_item_column_index"`
	Tag      string `json:"s_item_column_tag"`
	Value    string `json:"s_item_column_value"`
	Width    string `json:"s_item_column_width"`
	Wrap     string `json:"s_item_column_wrap"`
}

type TranslateMeta struct {
	Report string `json:"s_translate_report"`
	Seq    string `json:"s_translate_report_seq"`
}

type Preview struct {
	Headers     []HeaderField
	Columns     []Column
	HeaderBands []HeaderBand
	SubColumns  []Column
	Rows        []Row
	DebugLogs   []DebugLog
}

type HeaderField struct {
	Caption string
	Value   string
}

type Column struct {
	Key      string
	Caption  string
	Datatype string
	Digit    string
	Format   string
	Width    string
}

func (c Column) AlignClass() string {
	if c.Datatype == "N" {
		return "number"
	}
	return "text"
}

type HeaderBand struct {
	Caption string
	ColSpan int
	RowSpan int
}

type DebugLog struct {
	Level   string
	Kind    string
	Field   string
	Rows    string
	Message string
}
