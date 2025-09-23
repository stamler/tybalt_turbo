package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

// time_types id for Regular (R) in test DB: sdyfl3q7j7ap849

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
