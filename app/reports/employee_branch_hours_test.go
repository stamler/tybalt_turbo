package reports

import (
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
)

func TestQuoteSQLiteIdentifier(t *testing.T) {
	got := quoteSQLiteIdentifier(`North "Office"`)
	want := `"North ""Office"""`
	if got != want {
		t.Fatalf("quoteSQLiteIdentifier() = %q, want %q", got, want)
	}
}

func TestEmployeeBranchHoursCSVPreservesColumnsAndEscapesText(t *testing.T) {
	report := EmployeeBranchHoursReport{
		Columns: []string{"payrollId", "North, Office", "total"},
		Rows: []map[string]any{
			{
				"payrollId":     `ID "7"`,
				"North, Office": 1.5,
				"total":         1.5,
			},
		},
	}

	text, err := employeeBranchHoursCSV(report)
	if err != nil {
		t.Fatalf("employeeBranchHoursCSV() error = %v", err)
	}
	records, err := csv.NewReader(strings.NewReader(text)).ReadAll()
	if err != nil {
		t.Fatalf("failed to read generated CSV: %v", err)
	}
	want := [][]string{
		{"payrollId", "North, Office", "total"},
		{`ID "7"`, "1.5", "1.5"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("CSV records = %#v, want %#v", records, want)
	}
}

func TestEmployeeBranchHoursCSVIncludesHeaderForEmptyReport(t *testing.T) {
	report := EmployeeBranchHoursReport{
		Columns: []string{"payrollId", "total"},
		Rows:    []map[string]any{},
	}

	text, err := employeeBranchHoursCSV(report)
	if err != nil {
		t.Fatalf("employeeBranchHoursCSV() error = %v", err)
	}
	if text != "payrollId,total\n" {
		t.Fatalf("empty report CSV = %q, want header only", text)
	}
}
