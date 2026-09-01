package httpadapter

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"go_print_report_from_dataset/internal/adapters/renderer"
	"go_print_report_from_dataset/internal/domain/ports"
)

type Handler struct {
	parser        ports.DatasetParser
	renderer      ports.PreviewRenderer
	maxInputBytes int64
}

func NewHandler(parser ports.DatasetParser, previewRenderer ports.PreviewRenderer, maxInputBytes int64) *Handler {
	return &Handler{
		parser:        parser,
		renderer:      previewRenderer,
		maxInputBytes: maxInputBytes,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.index)
	mux.HandleFunc("POST /preview", h.preview)
	mux.HandleFunc("GET /healthz", h.healthz)
	return mux
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, indexHTML)
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxInputBytes)
	if err := r.ParseMultipartForm(h.maxInputBytes); err != nil {
		http.Error(w, fmt.Sprintf("parse form: %v", err), http.StatusBadRequest)
		return
	}

	input, err := h.readInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dataset, err := h.parser.Parse(bytes.NewReader(input))
	if err != nil {
		http.Error(w, fmt.Sprintf("parse dataset: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.Render(w, renderer.BuildPreview(dataset)); err != nil {
		http.Error(w, fmt.Sprintf("render preview: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, "ok\n")
}

func (h *Handler) readInput(r *http.Request) ([]byte, error) {
	if file, _, err := r.FormFile("dataset_file"); err == nil {
		defer file.Close()
		input, err := readLimited(file, h.maxInputBytes)
		if err != nil {
			return nil, fmt.Errorf("read upload: %w", err)
		}
		if len(bytes.TrimSpace(input)) > 0 {
			return input, nil
		}
	}

	input := []byte(r.FormValue("dataset_json"))
	if len(bytes.TrimSpace(input)) == 0 {
		return nil, fmt.Errorf("dataset input is required")
	}
	if int64(len(input)) > h.maxInputBytes {
		return nil, fmt.Errorf("dataset input exceeds %d bytes", h.maxInputBytes)
	}
	return input, nil
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	input, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(input)) > limit {
		return nil, fmt.Errorf("dataset input exceeds %d bytes", limit)
	}
	return input, nil
}

const indexHTML = `<!doctype html>
<html lang="th">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Report Dataset Preview</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Tahoma,sans-serif;margin:24px;color:#1f2933;background:#f7f7f4}
main{max-width:1100px;margin:0 auto}
form{display:grid;gap:14px}
textarea{width:100%;min-height:420px;box-sizing:border-box;font:13px ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;border:1px solid #cfd4dc;border-radius:4px;padding:12px;background:white;color:#1f2933}
input[type=file]{border:1px solid #cfd4dc;border-radius:4px;padding:8px;background:white}
button{justify-self:start;border:1px solid #8a8f6a;background:#596235;color:white;padding:9px 14px;border-radius:4px;cursor:pointer;font:inherit}
label{display:grid;gap:6px;font-weight:600}
span{font-size:13px;color:#4b5563;font-weight:400}
</style>
</head>
<body>
<main>
<form method="post" action="/preview" enctype="multipart/form-data">
<label>Dataset JSON or ZipToString base64 gzip
<textarea name="dataset_json" spellcheck="false"></textarea>
</label>
<label>Upload dataset file
<input type="file" name="dataset_file" accept=".json,.txt">
<span>Upload is used when both upload and pasted text are provided.</span>
</label>
<button type="submit">Preview</button>
</form>
</main>
</body>
</html>`
