package csvio

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

func TestNewReader_StripsLeadingBOM(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "with BOM", input: append(append([]byte{}, bom...), []byte("a,b\n1,2\n")...)},
		{name: "without BOM", input: []byte("a,b\n1,2\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(bytes.NewReader(tt.input), config.DefaultCSVOptions())

			record, err := r.Read()
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if len(record) == 0 || record[0] != "a" {
				t.Errorf("first field = %q, want %q (BOM should never leak into field data)", record, "a")
			}
		})
	}
}

func TestNewReader_UsesConfiguredDelimiter(t *testing.T) {
	opts := config.DefaultCSVOptions()
	opts.Delimiter = ';'
	r := NewReader(bytes.NewReader([]byte("a;b;c\n")), opts)

	record, err := r.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := []string{"a", "b", "c"}
	if len(record) != len(want) {
		t.Fatalf("record = %v, want %v", record, want)
	}
	for i, field := range want {
		if record[i] != field {
			t.Errorf("record[%d] = %q, want %q", i, record[i], field)
		}
	}
}

func TestReader_Read_ReturnsRecordsThenEOF(t *testing.T) {
	r := NewReader(bytes.NewReader([]byte("a,b\n1,2\n3,4\n")), config.DefaultCSVOptions())

	var got [][]string
	for {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		got = append(got, record)
	}

	want := [][]string{{"a", "b"}, {"1", "2"}, {"3", "4"}}
	if len(got) != len(want) {
		t.Fatalf("got %d records, want %d: %v", len(got), len(want), got)
	}
	for i, rec := range want {
		for j, field := range rec {
			if got[i][j] != field {
				t.Errorf("record[%d][%d] = %q, want %q", i, j, got[i][j], field)
			}
		}
	}
}

func TestReader_Read_ToleratesRaggedRecords(t *testing.T) {
	// The first record has 3 fields; the second has only 1. encoding/csv
	// would reject this by default (ErrFieldCount) unless
	// FieldsPerRecord is set to -1, which NewReader does.
	r := NewReader(bytes.NewReader([]byte("a,b,c\n1\n")), config.DefaultCSVOptions())

	first, err := r.Read()
	if err != nil {
		t.Fatalf("Read() first record error = %v", err)
	}
	if len(first) != 3 {
		t.Fatalf("first record = %v, want 3 fields", first)
	}

	second, err := r.Read()
	if err != nil {
		t.Fatalf("Read() second (ragged) record error = %v", err)
	}
	if len(second) != 1 || second[0] != "1" {
		t.Errorf("second record = %v, want [1]", second)
	}
}

func TestReadAll_StripsBOMAndToleratesRaggedRecords(t *testing.T) {
	input := append(append([]byte{}, bom...), []byte("a,b,c\n1,2,3\n4\n")...)

	got, err := ReadAll(bytes.NewReader(input), config.DefaultCSVOptions())
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	want := [][]string{{"a", "b", "c"}, {"1", "2", "3"}, {"4"}}
	if len(got) != len(want) {
		t.Fatalf("got %d records, want %d: %v", len(got), len(want), got)
	}
	if got[0][0] != "a" {
		t.Errorf("first field = %q, want %q (BOM should not leak into field data)", got[0][0], "a")
	}
	for i, rec := range want {
		if len(got[i]) != len(rec) {
			t.Errorf("record[%d] = %v, want %v", i, got[i], rec)
			continue
		}
		for j, field := range rec {
			if got[i][j] != field {
				t.Errorf("record[%d][%d] = %q, want %q", i, j, got[i][j], field)
			}
		}
	}
}

func TestReadAll_UsesConfiguredDelimiter(t *testing.T) {
	opts := config.DefaultCSVOptions()
	opts.Delimiter = '\t'

	got, err := ReadAll(bytes.NewReader([]byte("a\tb\n1\t2\n")), opts)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	want := [][]string{{"a", "b"}, {"1", "2"}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, rec := range want {
		for j, field := range rec {
			if got[i][j] != field {
				t.Errorf("record[%d][%d] = %q, want %q", i, j, got[i][j], field)
			}
		}
	}
}

func TestReadAll_EmptyInput_ReturnsEmptySlice(t *testing.T) {
	got, err := ReadAll(bytes.NewReader(nil), config.DefaultCSVOptions())
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}
