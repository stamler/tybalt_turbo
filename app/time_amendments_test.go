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
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
