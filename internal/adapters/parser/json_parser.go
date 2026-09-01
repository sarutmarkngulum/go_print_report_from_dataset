package parser

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go_print_report_from_dataset/internal/domain/report"
)

type JSONParser struct{}

func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

func (p *JSONParser) Parse(r io.Reader) (report.Dataset, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return report.Dataset{}, fmt.Errorf("read dataset: %w", err)
	}

	payload, err := decodePayload(bytes.TrimSpace(raw))
	if err != nil {
		return report.Dataset{}, err
	}

	var envelope struct {
		ReportHead map[string]any  `json:"report_head"`
		Items      *report.Items   `json:"items"`
		Extra      json.RawMessage `json:"-"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return report.Dataset{}, fmt.Errorf("parse dataset json: %w", err)
	}
	if envelope.Items == nil {
		return report.Dataset{}, fmt.Errorf("parse dataset: missing items")
	}
	if len(envelope.Items.ReportData) == 0 {
		return report.Dataset{}, fmt.Errorf("parse dataset: missing s_report_data")
	}

	return report.Dataset{
		ReportHead: envelope.ReportHead,
		Items:      *envelope.Items,
	}, nil
}

func decodePayload(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("parse dataset: empty input")
	}
	if raw[0] == '{' || raw[0] == '[' {
		return raw, nil
	}

	text := string(raw)
	var quoted string
	if err := json.Unmarshal(raw, &quoted); err == nil {
		text = quoted
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
	if err != nil {
		return raw, nil
	}

	zr, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return raw, nil
	}
	defer zr.Close()

	unzipped, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("decode gzip dataset: %w", err)
	}
	return bytes.TrimSpace(unzipped), nil
}
