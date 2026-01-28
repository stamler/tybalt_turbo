package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestJobAllocations_PutTransactionalUpdate(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const jobID = "cjf0kt0defhq480"
	const divisionID = "fy4i9poneukvq9u"

	scenarios := []tests.ApiScenario{
		{
			Name:   "update allocations for a job succeeds",
			Method: http.MethodPut,
			URL:    "/api/jobs/" + jobID,
			Body: strings.NewReader(`{
				"job": {
					"description": "Allocation update test"
				},
				"allocations": [
					{ "division": "` + divisionID + `", "hours": 10 }
				]
			}`),
			Headers:         map[string]string{"Authorization": recordToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`{"id":"` + jobID + `"}`},
			ExpectedEvents:  map[string]int{},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:            "job details includes allocations",
			Method:          http.MethodGet,
			URL:             "/api/jobs/" + jobID + "/details",
			Headers:         map[string]string{"Authorization": recordToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"allocations":[`, `"hours":10`},
			TestAppFactory:  testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// jobs.number: prevent changing the job number after creation (collection updateRule)
func TestJobsUpdate_NumberChangeBlockedByUpdateRule(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "attempting to change job number is blocked with 404",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"number": "99-9999"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				// updateRule blocks the request before reaching hooks
				"OnRecordUpdateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// jobs: allow updating fields when number is unchanged (control test)
func TestJobsUpdate_NumberUnchanged_AllowsOtherFieldUpdates(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	newDescription := "Updated description control"

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating description succeeds when number is not provided",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"description": "` + newDescription + `"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"description":"` + newDescription + `"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// jobs: when authorizing_document is PO, client_po is required (min length > 2)
func TestJobsUpdate_AuthorizingDocumentPO_Validations(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "PO set but missing client_po fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
                "authorizing_document": "PO",
                "client_po": ""
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"client_po":{"code":"client_po_min_length"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "PO set with short trimmed client_po fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
                "authorizing_document": "PO",
                "client_po": " 12 "
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"client_po":{"code":"client_po_min_length"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "PO set with valid trimmed client_po succeeds",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
                "authorizing_document": "PO",
                "client_po": "  ABC123  "
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"authorizing_document":"PO"`,
				`"client_po":"ABC123"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// jobs: proposal reference validation - Active proposals should trigger "proposal_not_awarded" error
func TestJobsCreate_ProposalReferenceValidation(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const activeRateSheet = "c41ofep525bcacj"

	scenarios := []tests.ApiScenario{
		{
			Name:   "creating job with Active proposal reference fails with proposal_not_awarded",
			Method: http.MethodPost,
			URL:    "/api/collections/jobs/records",
			Body: strings.NewReader(`{
				"description": "Test job with active proposal",
				"client": "ee3xvodl583b61o",
				"contact": "235g6k01xx3sdjk",
				"manager": "f2j5a8vk006baub",
				"authorizing_document": "Unauthorized",
				"branch": "80875lm27v8wgi4",
				"location": "87Q8H976+2M",
				"project_award_date": "2025-02-01",
				"proposal": "pactprop0000001",
				"rate_sheet": "` + activeRateSheet + `"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"proposal_not_awarded"`,
				`"message":"proposal must be set to Awarded"`,
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

// TestJobsCreateViaAPI_NumberAssigned verifies that creating a job via /api/jobs POST
// assigns a job number correctly (this was the bug we're fixing)
func TestJobsCreateViaAPI_NumberAssigned(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const divisionID = "fy4i9poneukvq9u"
	const activeRateSheet = "c41ofep525bcacj"

	scenarios := []tests.ApiScenario{
		{
			Name:   "creating job via /api/jobs POST assigns a job number",
			Method: http.MethodPost,
			URL:    "/api/jobs",
			Body: strings.NewReader(`{
				"job": {
					"description": "Test job via API",
					"client": "ee3xvodl583b61o",
					"contact": "235g6k01xx3sdjk",
					"manager": "f2j5a8vk006baub",
					"authorizing_document": "Unauthorized",
					"branch": "80875lm27v8wgi4",
					"location": "87Q8H976+2M",
					"project_award_date": "2025-02-01",
					"rate_sheet": "` + activeRateSheet + `"
				},
				"allocations": [
					{ "division": "` + divisionID + `", "hours": 10 }
				]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestJobsCreateViaAPI_ValidationErrors verifies that validation errors from
// ProcessJobCore are properly returned when creating via /api/jobs POST
func TestJobsCreateViaAPI_ValidationErrors(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const activeRateSheet = "c41ofep525bcacj"

	scenarios := []tests.ApiScenario{
		{
			Name:   "creating job without location fails validation",
			Method: http.MethodPost,
			URL:    "/api/jobs",
			Body: strings.NewReader(`{
				"job": {
					"description": "Test job without location",
					"client": "ee3xvodl583b61o",
					"contact": "235g6k01xx3sdjk",
					"manager": "f2j5a8vk006baub",
					"authorizing_document": "Unauthorized",
					"branch": "80875lm27v8wgi4",
					"location": "",
					"project_award_date": "2025-02-01",
					"rate_sheet": "` + activeRateSheet + `"
				},
				"allocations": []
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"location":{"code":"invalid_or_missing"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "creating job with invalid date configuration fails",
			Method: http.MethodPost,
			URL:    "/api/jobs",
			Body: strings.NewReader(`{
				"job": {
					"description": "Test job with invalid dates",
					"client": "ee3xvodl583b61o",
					"contact": "235g6k01xx3sdjk",
					"manager": "f2j5a8vk006baub",
					"authorizing_document": "Unauthorized",
					"branch": "80875lm27v8wgi4",
					"location": "87Q8H976+2M",
					"project_award_date": "2025-02-01",
					"proposal_opening_date": "2025-02-01",
					"rate_sheet": "` + activeRateSheet + `"
				},
				"allocations": []
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"project_award_date":{"code":"invalid_date_configuration"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestJobsUpdateViaAPI_ValidationErrors verifies that validation errors from
// ProcessJobCore are properly returned when updating via /api/jobs/{id} PUT
func TestJobsUpdateViaAPI_ValidationErrors(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const jobID = "cjf0kt0defhq480"
	const divisionID = "fy4i9poneukvq9u"

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating job to invalid status fails validation",
			Method: http.MethodPut,
			URL:    "/api/jobs/" + jobID,
			Body: strings.NewReader(`{
				"job": {
					"status": "Awarded"
				},
				"allocations": [
					{ "division": "` + divisionID + `", "hours": 10 }
				]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"status":{"code":"invalid_status_for_type"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestJobsCreate_InactiveManagerRejected verifies that setting a manager or
// alternate_manager to an inactive user (admin_profiles.active=false) is rejected
func TestJobsCreate_InactiveManagerRejected(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// u_inactive is a test user with admin_profiles.active=0
	const inactiveUserID = "u_inactive"
	const activeManagerID = "f2j5a8vk006baub"
	const activeRateSheet = "c41ofep525bcacj"

	scenarios := []tests.ApiScenario{
		{
			Name:   "creating job with inactive manager fails",
			Method: http.MethodPost,
			URL:    "/api/jobs",
			Body: strings.NewReader(`{
				"job": {
					"description": "Test job with inactive manager",
					"client": "ee3xvodl583b61o",
					"contact": "235g6k01xx3sdjk",
					"manager": "` + inactiveUserID + `",
					"authorizing_document": "Unauthorized",
					"branch": "80875lm27v8wgi4",
					"location": "87Q8H976+2M",
					"project_award_date": "2025-02-01",
					"rate_sheet": "` + activeRateSheet + `"
				},
				"allocations": []
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"manager":{"code":"manager_not_active"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "creating job with inactive alternate_manager fails",
			Method: http.MethodPost,
			URL:    "/api/jobs",
			Body: strings.NewReader(`{
				"job": {
					"description": "Test job with inactive alternate manager",
					"client": "ee3xvodl583b61o",
					"contact": "235g6k01xx3sdjk",
					"manager": "` + activeManagerID + `",
					"alternate_manager": "` + inactiveUserID + `",
					"authorizing_document": "Unauthorized",
					"branch": "80875lm27v8wgi4",
					"location": "87Q8H976+2M",
					"project_award_date": "2025-02-01",
					"rate_sheet": "` + activeRateSheet + `"
				},
				"allocations": []
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"alternate_manager":{"code":"alternate_manager_not_active"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestJobsAPI_InactiveDivisionFails verifies that creating or updating jobs with
// inactive divisions in allocations fails validation.
//
// Test data:
//   - apkev2ow1zjtm7w is an inactive division
//   - fy4i9poneukvq9u is an active division
//   - cjf0kt0defhq480 is an existing job (25-001)
func TestJobsAPI_InactiveDivisionFails(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const activeDivision = "fy4i9poneukvq9u"
	const inactiveDivision = "apkev2ow1zjtm7w"
	const existingJobID = "cjf0kt0defhq480"
	const activeRateSheet = "c41ofep525bcacj"

	scenarios := []tests.ApiScenario{
		{
			Name:   "creating job with inactive division in allocations fails",
			Method: http.MethodPost,
			URL:    "/api/jobs",
			Body: strings.NewReader(`{
				"job": {
					"description": "Test job with inactive division",
					"client": "ee3xvodl583b61o",
					"contact": "235g6k01xx3sdjk",
					"manager": "f2j5a8vk006baub",
					"authorizing_document": "Unauthorized",
					"branch": "80875lm27v8wgi4",
					"location": "87Q8H976+2M",
					"project_award_date": "2025-02-01",
					"rate_sheet": "` + activeRateSheet + `"
				},
				"allocations": [
					{ "division": "` + inactiveDivision + `", "hours": 10 }
				]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"division_not_active"`,
				`"error":"division is inactive:`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "creating job with active division in allocations succeeds",
			Method: http.MethodPost,
			URL:    "/api/jobs",
			Body: strings.NewReader(`{
				"job": {
					"description": "Test job with active division",
					"client": "ee3xvodl583b61o",
					"contact": "235g6k01xx3sdjk",
					"manager": "f2j5a8vk006baub",
					"authorizing_document": "Unauthorized",
					"branch": "80875lm27v8wgi4",
					"location": "87Q8H976+2M",
					"project_award_date": "2025-02-01",
					"rate_sheet": "` + activeRateSheet + `"
				},
				"allocations": [
					{ "division": "` + activeDivision + `", "hours": 10 }
				]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "updating job with inactive division in allocations fails",
			Method: http.MethodPut,
			URL:    "/api/jobs/" + existingJobID,
			Body: strings.NewReader(`{
				"job": {},
				"allocations": [
					{ "division": "` + inactiveDivision + `", "hours": 5 }
				]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"division_not_active"`,
				`"error":"division is inactive:`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "updating job with active division in allocations succeeds",
			Method: http.MethodPut,
			URL:    "/api/jobs/" + existingJobID,
			Body: strings.NewReader(`{
				"job": {},
				"allocations": [
					{ "division": "` + activeDivision + `", "hours": 5 }
				]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"` + existingJobID + `"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
