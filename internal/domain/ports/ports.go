package ports

import (
	"io"

	"go_print_report_from_dataset/internal/domain/report"
)

type DatasetParser interface {
	Parse(r io.Reader) (report.Dataset, error)
}

type PreviewRenderer interface {
	Render(w io.Writer, preview report.Preview) error
}
