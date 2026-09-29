package routes

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	_ "modernc.org/sqlite"
)

// Run against an isolated snapshot. This benchmark opens it read-only and never
// starts PocketBase, applies migrations, or prints customer identifiers.
func BenchmarkWIPReports(b *testing.B) {
	path := os.Getenv("WIP_BENCHMARK_DB")
	if path == "" {
		b.Skip("set WIP_BENCHMARK_DB to an isolated database snapshot")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		b.Fatal(err)
	}
	uri := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}
	db, err := dbx.Open("sqlite", uri.String())
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	db.DB().SetMaxOpenConns(1)
	var manager, branch, division string
	for _, selection := range []struct {
		query string
		value *string
	}{
		{`SELECT manager FROM jobs WHERE status = 'Active' AND number NOT LIKE 'P%' AND project_value > 0 GROUP BY manager ORDER BY COUNT(*) DESC, manager LIMIT 1`, &manager},
		{`SELECT branch FROM jobs WHERE status = 'Active' AND number NOT LIKE 'P%' AND project_value > 0 GROUP BY branch ORDER BY COUNT(*) DESC, branch LIMIT 1`, &branch},
		{`SELECT a.division FROM job_time_allocations a JOIN jobs j ON j.id = a.job WHERE j.status = 'Active' AND j.number NOT LIKE 'P%' AND j.project_value > 0 GROUP BY a.division ORDER BY COUNT(*) DESC, a.division LIMIT 1`, &division},
	} {
		if err := db.NewQuery(selection.query).Row(selection.value); err != nil {
			b.Fatal(err)
		}
	}
	asOf := os.Getenv("WIP_BENCHMARK_AS_OF")
	if asOf == "" {
		asOf = time.Now().UTC().Format("2006-01-02")
	}
	for _, tc := range []struct {
		name, mode, uid, branch, division string
		requireTime                       bool
	}{
		{"MyEmpty", "my", "__wip_benchmark_no_user__", "", "", true},
		{"MyLargest", "my", manager, "", "", true},
		{"BranchAll", "branch", "", "", "", true},
		{"BranchLargest", "branch", "", branch, "", true},
		{"DivisionHours", "division", "", "", division, true},
		{"DivisionAll", "division", "", "", division, false},
	} {
		b.Run(tc.name, func(b *testing.B) {
			params := dbx.Params{"mode": tc.mode, "uid": tc.uid, "branch": tc.branch, "division": tc.division,
				"require_time": tc.requireTime, "start_date": "", "end_date": asOf}
			b.ReportAllocs()
			for b.Loop() {
				result := WIPReport{Items: []WIPReportRow{}, AsOf: asOf}
				if err := loadWIPReport(db, params, &result); err != nil {
					b.Fatal(err)
				}
				if _, err := json.Marshal(result); err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(result.Items)), "rows/op")
			}
		})
	}
}
