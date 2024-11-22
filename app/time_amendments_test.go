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
			Url:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": true,
				"week_ending": "2006-01-02"
				}`),
			RequestHeaders: map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"committed":""`,
				`"week_ending":"2024-09-07"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         1,
				"OnModelAfterCreate":          1,
				"OnRecordBeforeCreateRequest": 1,
				"OnRecordAfterCreateRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid time_amendment fails creation when a corresponding time_sheets record cannot be found and skip_tsid_check is false",
			Method: http.MethodPost,
			Url:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": false,
				"week_ending": "2006-01-02"
				}`),
			RequestHeaders: map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"global":{"code":"no_time_sheet"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 1,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "a time_amendment's creator property must match the authenticated user's id",
			Method: http.MethodPost,
			Url:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "tqqf7q0f3378rvp",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": true,
				"week_ending": "2006-01-02"
				}`),
			RequestHeaders: map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"creator":{"code":"creator_mismatch"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 1,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting the committed property is forbidden",
			Method: http.MethodPost,
			Url:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"committed": "2024-11-01 00:00:00",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": true,
				"week_ending": "2006-01-02"
				}`),
			RequestHeaders: map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":400,"message":"Failed to create record."`,
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
			Url:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"committer": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"description": "test time_amendment",
				"hours": 1,
				"skip_tsid_check": true,
				"week_ending": "2006-01-02"
				}`),
			RequestHeaders: map[string]string{"Authorization": creatorToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":400,"message":"Failed to create record."`,
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
			Url:             "/api/time_amendments/qn4jyrkxp3pfjom/commit",
			RequestHeaders:  map[string]string{"Authorization": commitToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"message":"Record committed successfully"`},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 1,
				"OnModelAfterUpdate":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller without the commit claim cannot commit time_amendments records",
			Method:          http.MethodPost,
			Url:             "/api/time_amendments/qn4jyrkxp3pfjom/commit",
			RequestHeaders:  map[string]string{"Authorization": nonCommitToken},
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
