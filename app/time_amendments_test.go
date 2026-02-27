// time_amendments_test.go
package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestTimeAmendmentsCreate(t *testing.T) {
	creatorToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "valid time_amendment gets a correct week_ending and bypasses tsid check if skip_tsid_check is true",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": true,
				"week_ending": "2006-01-02"
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"committed":""`,
				`"week_ending":"2024-09-07"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time_amendment allows inactive division (retroactive)",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "apkev2ow1zjtm7w",
				"branch": "80875lm27v8wgi4",
				"description": "inactive division allowed",
				"hours": 1,
				"skip_tsid_check": true
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"division":"apkev2ow1zjtm7w"`,
				`"week_ending":"2024-09-07"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid time_amendment fails creation when a corresponding time_sheets record cannot be found and skip_tsid_check is false",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": false,
				"week_ending": "2006-01-02"
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"global":{"code":"no_time_sheet"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
				"OnRecordCreate":        0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time_amendment with job fails when division is not allocated to that job",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "90drdtwx5v4ew70",
				"job": "test_job_w_rs",
				"branch": "80875lm27v8wgi4",
				"description": "job amendment with wrong division",
				"hours": 1,
				"skip_tsid_check": true
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"division":{"code":"division_not_allowed"`,
				`Division BM is not allocated to this job`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
				"OnRecordCreate":        0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time_amendment with invalid job fails with invalid reference",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"job": "notarealjobid123",
				"branch": "80875lm27v8wgi4",
				"description": "job amendment with invalid job",
				"hours": 1,
				"skip_tsid_check": true
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"job":{"code":"invalid_reference"`,
				`Invalid job reference.`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
				"OnRecordCreate":        0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time_amendment with closed job fails with not active",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"job": "zke3cs3yipplwtu",
				"branch": "80875lm27v8wgi4",
				"description": "job amendment with closed job",
				"hours": 1,
				"skip_tsid_check": true
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"job":{"code":"not_active"`,
				`Job status must be Active`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
				"OnRecordCreate":        0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "creator is server-stamped to the authenticated user",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "tqqf7q0f3378rvp",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": true,
				"week_ending": "2006-01-02"
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"creator":"f2j5a8vk006baub"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
				"OnRecordCreate":        1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting the committed property is forbidden",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"committed": "2024-11-01 00:00:00",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": true,
				"week_ending": "2006-01-02"
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Failed to create record.","status":400`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 0,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting the committer property is forbidden",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"committer": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": true,
				"week_ending": "2006-01-02"
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Failed to create record.","status":400`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 0,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestTimeAmendmentsUpdate(t *testing.T) {
	creatorToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "a requester can update a time_amendments record they created",
			Method: http.MethodPatch,
			URL:    "/api/collections/time_amendments/records/qn4jyrkxp3pfjom",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "f2j5a8vk006baub",
				"date": "2024-09-25",
				"division": "fy4i9poneukvq9u",
				"branch": "80875lm27v8wgi4",
				"description": "test time_amendment",
				"hours": 1,
				"job": "u09fwwcg07y03m7",
				"tsid": "j1lr2oddjongtoj",
				"week_ending": "2006-01-02"
				}`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"committed":""`,
				`"week_ending":"2024-09-28"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// Any `tame` claim holder may edit uncommitted time amendments by policy.

		// TODO: write more tests
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestTimeAmendmentsRoutes(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}
	nonCommitToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:            "caller with the commit claim can commit time_amendments records",
			Method:          http.MethodPost,
			URL:             "/api/time_amendments/qn4jyrkxp3pfjom/commit",
			Headers:         map[string]string{"Authorization": commitToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"message":"Record committed successfully"`},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller without the commit claim cannot commit time_amendments records",
			Method:          http.MethodPost,
			URL:             "/api/time_amendments/qn4jyrkxp3pfjom/commit",
			Headers:         map[string]string{"Authorization": nonCommitToken},
			ExpectedStatus:  403,
			ExpectedContent: []string{`"error":"You are not authorized to commit this record."`},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 0,
				"OnModelAfterUpdate":  0,
				"OnBeforeApiError":    0,
				"OnAfterApiError":     0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestTimeAmendmentsBranchRequired(t *testing.T) {
	creatorToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "create succeeds when branch is provided",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
                "creator": "f2j5a8vk006baub",
                "time_type": "sdyfl3q7j7ap849",
                "uid": "rzr98oadsp9qc11",
                "date": "2024-09-02",
                "division": "vccd5fo56ctbigh",
                "branch": "80875lm27v8wgi4",
                "description": "branch present",
                "hours": 1,
                "skip_tsid_check": true
            }`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"80875lm27v8wgi4"`,
				`"week_ending":"2024-09-07"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "create fails when branch is missing",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
                "creator": "f2j5a8vk006baub",
                "time_type": "sdyfl3q7j7ap849",
                "uid": "rzr98oadsp9qc11",
                "date": "2024-09-02",
                "division": "vccd5fo56ctbigh",
                "description": "branch missing",
                "branch": "",
                "hours": 1,
                "skip_tsid_check": true
            }`),
			Headers:        map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"branch":{"code":"validation_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
