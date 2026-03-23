package reports

import (
	"strings"
	"testing"
)

func TestPayablesRowToRecordUsesRecordDateForDisplayColumns(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		recordDate   string
		approvalDate string
		wantDay      string
		wantMonth    string
		wantYear     string
	}{
		{
			name:         "sqlite zulu with milliseconds",
			recordDate:   "2024-09-01",
			approvalDate: "2026-03-11 15:04:05.000Z",
			wantDay:      "1",
			wantMonth:    "Sep",
			wantYear:     "2024",
		},
		{
			name:         "sqlite utc suffix",
			recordDate:   "2024-09-02",
			approvalDate: "2026-03-11 15:04:05.000 +0000 UTC",
			wantDay:      "2",
			wantMonth:    "Sep",
			wantYear:     "2024",
		},
		{
			name:         "rfc3339 zulu",
			recordDate:   "2024-09-03",
			approvalDate: "2026-03-11 15:04:05Z",
			wantDay:      "3",
			wantMonth:    "Sep",
			wantYear:     "2024",
		},
		{
			name:         "date only",
			recordDate:   "2024-09-04",
			approvalDate: "2026-03-11",
			wantDay:      "4",
			wantMonth:    "Sep",
			wantYear:     "2024",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := payablesRow{
				PaymentType:  "Expense",
				JobNumber:    "24-321",
				DivisionCode: "BM",
				BranchCode:   "TOR",
				POType:       "Recurring",
				RecordDate:   tc.recordDate,
				ApprovalDate: tc.approvalDate,
				Total:        "123.45",
				PONumber:     "2603-0001",
				Description:  "Fixture row",
				VendorName:   "Vendor Name",
				Employee:     "Test User",
				ApprovedBy:   "Approver Name",
				Status:       "Active",
			}

			record := row.toRecord()
			if got := record[4]; got != "Recurring" {
				t.Fatalf("type = %q, want %q", got, "Recurring")
			}
			if got := record[5]; got != tc.wantDay {
				t.Fatalf("day = %q, want %q", got, tc.wantDay)
			}
			if got := record[6]; got != tc.wantMonth {
				t.Fatalf("month = %q, want %q", got, tc.wantMonth)
			}
			if got := record[7]; got != tc.wantYear {
				t.Fatalf("year = %q, want %q", got, tc.wantYear)
			}
			if got := record[16]; got != "Fixture row" {
				t.Fatalf("description = %q, want %q", got, "Fixture row")
			}
			if got := record[19]; got != "Approver Name" {
				t.Fatalf("approved by = %q, want %q", got, "Approver Name")
			}
			if got := record[20]; got != "TURBO" {
				t.Fatalf("entered by = %q, want %q", got, "TURBO")
			}
		})
	}
}

func TestRowsToCSVAndTSV(t *testing.T) {
	t.Parallel()

	rows := []payablesRow{
		{
			PaymentType:  "Expense",
			JobNumber:    "24-321",
			DivisionCode: "BM",
			BranchCode:   "TOR",
			POType:       "One-Time",
			RecordDate:   "2024-09-01",
			ApprovalDate: "2026-03-11",
			Total:        "123.45",
			PONumber:     "2603-0001",
			Description:  `Fixture "quoted", row`,
			VendorName:   "Vendor Name",
			Employee:     "Test User",
			ApprovedBy:   "Approver Name",
			Status:       "Active",
		},
	}

	csvString, err := rowsToCSV(rows)
	if err != nil {
		t.Fatalf("rowsToCSV returned error: %v", err)
	}

	if !strings.HasPrefix(csvString, strings.Join(payablesSpreadsheetHeaders, ",")+"\n") {
		t.Fatalf("csv header missing, got %q", csvString)
	}
	if !strings.Contains(csvString, `"Fixture ""quoted"", row"`) {
		t.Fatalf("csv should escape quotes and commas, got %q", csvString)
	}

	tsvString := rowsToTSV(rows)
	if strings.Contains(tsvString, "Acct/Visa/Exp\tJob #") {
		t.Fatalf("tsv should not include headers, got %q", tsvString)
	}
	if !strings.Contains(tsvString, "Expense\t24-321\tBM\tTOR\tOne-Time\t1\tSep\t2024") {
		t.Fatalf("tsv should contain tab-separated row data, got %q", tsvString)
	}
	if !strings.HasSuffix(tsvString, "\n") {
		t.Fatalf("tsv should end with newline, got %q", tsvString)
	}
}
