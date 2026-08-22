package csvio

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strings"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

func TestWriter_UseBOM_ControlsBOMPresence(t *testing.T) {
	tests := []struct {
		name   string
		useBOM bool
	}{
		{name: "BOM enabled", useBOM: true},
		{name: "BOM disabled", useBOM: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := config.DefaultCSVOptions()
			opts.UseBOM = tt.useBOM

			w, err := NewWriter(&buf, opts)
			if err != nil {
				t.Fatalf("NewWriter() error = %v", err)
			}
			if err := w.WriteRecord([]string{"a", "b"}); err != nil {
				t.Fatalf("WriteRecord() error = %v", err)
			}
			if err := w.Flush(); err != nil {
				t.Fatalf("Flush() error = %v", err)
			}

			hasBOM := bytes.HasPrefix(buf.Bytes(), bom)
			if hasBOM != tt.useBOM {
				t.Errorf("hasBOM = %v, want %v", hasBOM, tt.useBOM)
			}
		})
	}
}

func TestWriter_Delimiter_UsesConfiguredSeparator(t *testing.T) {
	var buf bytes.Buffer
	opts := config.DefaultCSVOptions()
	opts.UseBOM = false
	opts.Delimiter = ';'

	w, err := NewWriter(&buf, opts)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}
	if err := w.WriteRecord([]string{"a", "b", "c"}); err != nil {
		t.Fatalf("WriteRecord() error = %v", err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if !strings.Contains(buf.String(), "a;b;c") {
		t.Errorf("output = %q, want it to contain %q", buf.String(), "a;b;c")
	}
}

func TestWriter_LineEnding_ControlsTerminator(t *testing.T) {
	tests := []struct {
		name       string
		lineEnding string
		wantCRLF   bool
	}{
		{name: "CRLF", lineEnding: "\r\n", wantCRLF: true},
		{name: "LF", lineEnding: "\n", wantCRLF: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := config.DefaultCSVOptions()
			opts.UseBOM = false
			opts.LineEnding = tt.lineEnding

			w, err := NewWriter(&buf, opts)
			if err != nil {
				t.Fatalf("NewWriter() error = %v", err)
			}
			if err := w.WriteRecord([]string{"a"}); err != nil {
				t.Fatalf("WriteRecord() error = %v", err)
			}
			if err := w.Flush(); err != nil {
				t.Fatalf("Flush() error = %v", err)
			}

			hasCRLF := strings.HasSuffix(buf.String(), "\r\n")
			if hasCRLF != tt.wantCRLF {
				t.Errorf("output = %q, hasCRLF = %v, want %v", buf.String(), hasCRLF, tt.wantCRLF)
			}
		})
	}
}

type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

func TestNewWriter_PropagatesBOMWriteError(t *testing.T) {
	wantErr := errors.New("disk full")
	opts := config.DefaultCSVOptions()
	opts.UseBOM = true

	_, err := NewWriter(errWriter{err: wantErr}, opts)
	if !errors.Is(err, wantErr) {
		t.Errorf("NewWriter() error = %v, want %v", err, wantErr)
	}
}

func TestWriter_RoundTrip_ReadableByEncodingCSV(t *testing.T) {
	var buf bytes.Buffer
	opts := config.DefaultCSVOptions()

	w, err := NewWriter(&buf, opts)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	records := [][]string{
		{"id", "name", "note"},
		{"1", "Alice", "has, a comma"},
		{"2", "Bob", "has \"quotes\""},
	}
	for _, rec := range records {
		if err := w.WriteRecord(rec); err != nil {
			t.Fatalf("WriteRecord() error = %v", err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Strip the BOM before decoding, since encoding/csv does not do it
	// automatically; readers built on top of this package are
	// responsible for that (see excel/csvio.NewReader).
	data := bytes.TrimPrefix(buf.Bytes(), bom)
	reader := csv.NewReader(bytes.NewReader(data))
	got, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(got) != len(records) {
		t.Fatalf("got %d records, want %d", len(got), len(records))
	}
	for i, rec := range records {
		for j, field := range rec {
			if got[i][j] != field {
				t.Errorf("record[%d][%d] = %q, want %q", i, j, got[i][j], field)
			}
		}
	}
}
