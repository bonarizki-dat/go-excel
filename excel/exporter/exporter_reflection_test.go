package exporter

import (
	"context"
	"strings"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

type reflectionPerson struct {
	Name   string
	hidden int
	Skip   string `excel:"-"`
}

type ReflectionBase struct {
	Name string
}

type reflectionNamed struct {
	ReflectionBase
	Age int
}

func TestExporter_PrepareData_SliceOfStruct(t *testing.T) {
	data := []reflectionPerson{
		{Name: "Ada", hidden: 1, Skip: "x"},
		{Name: "Bob", hidden: 2, Skip: "y"},
	}
	rows, err := NewExporter(data, excel.WithPreallocation(true)).prepareData(context.Background())
	if err != nil {
		t.Fatalf("prepareData: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if len(rows[0]) != 1 || rows[0][0] != "Ada" {
		t.Errorf("row0 = %v, want [Ada] (unexported and excel:\"-\" skipped)", rows[0])
	}
}

func TestExporter_PrepareData_ArrayOfStruct(t *testing.T) {
	data := [1]reflectionPerson{{Name: "Ada"}}
	rows, err := NewExporter(data).prepareData(context.Background())
	if err != nil {
		t.Fatalf("prepareData: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != "Ada" {
		t.Errorf("rows = %v, want [[Ada]]", rows)
	}
}

func TestExporter_PrepareData_PointerToStruct(t *testing.T) {
	data := &reflectionPerson{Name: "Ada"}
	rows, err := NewExporter(data).prepareData(context.Background())
	if err != nil {
		t.Fatalf("prepareData: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != "Ada" {
		t.Errorf("rows = %v, want [[Ada]]", rows)
	}
}

func TestExporter_PrepareData_EmbeddedStruct(t *testing.T) {
	data := reflectionNamed{
		ReflectionBase: ReflectionBase{Name: "Ada"},
		Age:            36,
	}
	rows, err := NewExporter(data).prepareData(context.Background())
	if err != nil {
		t.Fatalf("prepareData: %v", err)
	}
	if len(rows) != 1 || len(rows[0]) != 2 || rows[0][0] != "Ada" || rows[0][1] != 36 {
		t.Errorf("rows = %v, want [Ada 36]", rows)
	}
}

func TestExporter_PrepareData_ScalarFallback(t *testing.T) {
	rows, err := NewExporter(42).prepareData(context.Background())
	if err != nil {
		t.Fatalf("prepareData: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != 42 {
		t.Errorf("rows = %v, want [[42]]", rows)
	}
}

func TestExporter_PrepareData_SliceOfPointerToStruct(t *testing.T) {
	data := []*reflectionPerson{{Name: "Ada"}}
	rows, err := NewExporter(data).prepareData(context.Background())
	if err != nil {
		t.Fatalf("prepareData: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != "Ada" {
		t.Errorf("rows = %v, want [[Ada]]", rows)
	}
}

func TestExporter_PrepareData_NonStructSliceElement(t *testing.T) {
	_, err := NewExporter([]int{1, 2}).prepareData(context.Background())
	if err == nil || !strings.Contains(err.Error(), "expected struct") {
		t.Fatalf("error = %v, want expected struct", err)
	}
}
