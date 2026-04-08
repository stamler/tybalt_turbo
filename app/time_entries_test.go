package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
	"tybalt/hooks"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// time_types id for Regular (R) in test DB: sdyfl3q7j7ap849
// Test fixtures:
//   - users: "rzr98oadsp9qc11" (time@test.com)
//   - time_types: "sdyfl3q7j7ap849" (Regular)
//   - divisions: "fy4i9poneukvq9u" (active division)
//   - jobs: "test_job_w_rs" (98-8001, project WITH rate_sheet c41ofep525bcacj)
//   - jobs: "u09fwwcg07y03m7" (24-291, project with NO rate_sheet)
//   - rate_roles: "tbgoiwwwfj8cvju" (Principal)

func TestTimeEntriesCreate_InactiveDivisionFails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "otherwise valid time entry with Inactive division fails",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "apkev2ow1zjtm7w",
				"description": "test time entry",
				"hours": 1
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"division":{"code":"not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestTimeEntriesCreate_JobRequiresRole tests that a time entry for any job
// requires the role field to be set, regardless of whether the job has a rate_sheet.
func TestTimeEntriesCreate_JobRequiresRole(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time entry with job (has rate_sheet) but no role fails",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry for job with rate sheet",
				"hours": 1,
				"job": "test_job_w_rs"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"role":`,
				`Role is required when a job is assigned`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry with job (no rate_sheet) but no role fails",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry for job without rate sheet",
				"hours": 1,
				"job": "u09fwwcg07y03m7"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"role":`,
				`Role is required when a job is assigned`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry with job (has rate_sheet) and role succeeds",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry for job with rate sheet and role",
				"hours": 1,
				"job": "test_job_w_rs",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"role":"tbgoiwwwfj8cvju"`,
				`"job":"test_job_w_rs"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest":      1,
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelAfterCreateSuccess":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry with job (no rate_sheet) and role succeeds",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry for job without rate sheet but with role",
				"hours": 1,
				"job": "u09fwwcg07y03m7",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"role":"tbgoiwwwfj8cvju"`,
				`"job":"u09fwwcg07y03m7"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest":      1,
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelAfterCreateSuccess":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestTimeEntriesCreate_BranchResolutionMatchesPurchaseOrders(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "branch defaults from creator profile when omitted and no job is set",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "default branch assignment",
				"hours": 1
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"80875lm27v8wgi4"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest":      1,
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelAfterCreateSuccess":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "explicit branch survives when no job is set",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "manual branch assignment",
				"hours": 1,
				"branch": "xeq9q81q5307f70"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"xeq9q81q5307f70"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest":      1,
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelAfterCreateSuccess":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "job branch overrides explicit branch when job is set",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "job branch assignment",
				"hours": 1,
				"job": "test_job_w_rs",
				"role": "tbgoiwwwfj8cvju",
				"branch": "80875lm27v8wgi4"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"job":"test_job_w_rs"`,
				`"branch":"xeq9q81q5307f70"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest":      1,
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelAfterCreateSuccess":  1,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(tb)
				setJobBranch(tb, app, "test_job_w_rs", "xeq9q81q5307f70")
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestTimeEntriesCreate_InvalidJobBranchResolutionErrorsRemainFieldScoped(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "invalid job id returns job not_found error",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "invalid job branch lookup",
				"hours": 1,
				"job": "does_not_exist",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"hook error when cleaning time entry"`,
				`"job":{"code":"not_found","message":"referenced job not found"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "job without branch returns job missing_branch error",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "job missing branch",
				"hours": 1,
				"job": "test_job_w_rs",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"hook error when cleaning time entry"`,
				`"job":{"code":"missing_branch","message":"referenced job is missing a branch"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(tb)
				if _, err := app.DB().NewQuery("UPDATE jobs SET branch = '' WHERE id = {:id}").Bind(dbx.Params{
					"id": "test_job_w_rs",
				}).Execute(); err != nil {
					tb.Fatalf("failed to blank branch on job test_job_w_rs: %v", err)
				}
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestTimeEntriesCreate_DivisionNotAllocatedToJob tests that a time entry
// fails when the division is not allocated to the job via job_time_allocations.
// Test fixtures:
//   - jobs: "test_job_w_rs" (98-8001) has allocation for division "fy4i9poneukvq9u" (MD)
//   - jobs: "tt4eipt6wapu9zh" (24-334) has NO allocations
//   - division "90drdtwx5v4ew70" (BM) is NOT allocated to test_job_w_rs
func TestTimeEntriesCreate_DivisionNotAllocatedToJob(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time entry with division not allocated to job fails",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "90drdtwx5v4ew70",
				"description": "test time entry with wrong division for job",
				"hours": 1,
				"job": "test_job_w_rs",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"division":`,
				`Division BM is not allocated to this job`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry with allocated division succeeds",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry with correct division for job",
				"hours": 1,
				"job": "test_job_w_rs",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"division":"fy4i9poneukvq9u"`,
				`"job":"test_job_w_rs"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest":      1,
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelAfterCreateSuccess":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry with job that has no allocations fails",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "90drdtwx5v4ew70",
				"description": "test time entry for job without allocations",
				"hours": 1,
				"job": "tt4eipt6wapu9zh",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"job":`,
				`This job has no division allocations configured`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestTimeEntriesCreate_NoJob_ClearsRole tests that when no job is provided,
// the role field is cleared by the clean hook.
func TestTimeEntriesCreate_NoJob_ClearsRole(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time entry without job has role cleared",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry without job",
				"hours": 1,
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"role":""`,
				`"job":""`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest":      1,
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelAfterCreateSuccess":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestTimeEntriesCreate_AwardedProposalWindow(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	originalNow := hooks.TimeTrackingNow
	hooks.TimeTrackingNow = func() time.Time {
		return time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC)
	}
	defer func() {
		hooks.TimeTrackingNow = originalNow
	}()

	scenarios := []tests.ApiScenario{
		{
			Name:   "time entry with recently awarded proposal succeeds",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "time entry against recent awarded proposal",
				"hours": 1,
				"job": "awproprecent001",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"job":"awproprecent001"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest":      1,
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelAfterCreateSuccess":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry with stale awarded proposal fails",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "time entry against stale awarded proposal",
				"hours": 1,
				"job": "awpropstale0001",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"job":{"code":"proposal_awarded_more_than_30_days_ago"`,
				`Proposal was awarded more than 30 days in the past; use the referencing project instead`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry with awarded proposal without referencing project fails",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "time entry against awarded proposal without project",
				"hours": 1,
				"job": "awpropnoref0001",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"job":{"code":"proposal_awarded_without_referencing_project"`,
				`Proposal is Awarded but has no referencing project yet; you cannot charge time to the proposal`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry with future project award date fails",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "time entry against proposal with future project award date",
				"hours": 1,
				"job": "awproprecent001",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"job":{"code":"invalid_project_award_date"`,
				`Referenced project has a future project award date`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				t.Helper()

				project, err := app.FindRecordById("jobs", "awprojrecent001")
				if err != nil {
					t.Fatalf("failed to load referencing project: %v", err)
				}

				project.Set("project_award_date", "2026-03-25")
				if err := app.Save(project); err != nil {
					t.Fatalf("failed to update referencing project: %v", err)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
