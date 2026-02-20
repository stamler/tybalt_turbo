package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"mime/multipart"
	"net/http"
	"testing"
	"time"
	"tybalt/internal/testutils"
	"tybalt/routes"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func ensureDefaultPOKind(payload map[string]any) {
	if _, exists := payload["kind"]; !exists {
		payload["kind"] = utilities.DefaultExpenditureKindID()
	}
}

func TestPurchaseOrdersCreate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for user with po_approver claim
	poApproverToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for user with division-specific po_approver claim
	divisionApproverToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for user with po_approver_tier3 claim
	po_approver_tier3Token, err := testutils.GenerateRecordToken("users", "hal@2005.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for user with po_approver_tier2 claim
	po_approver_tier2Token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get approval tier values once and reuse them throughout the tests
	app := testutils.SetupTestApp(t)
	tier1, tier2 := testutils.GetApprovalTiers(app)
	sponsorshipKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "sponsorship",
	})
	if err != nil {
		t.Fatalf("failed to load sponsorship expenditure kind: %v", err)
	}
	standardKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "standard",
	})
	if err != nil {
		t.Fatalf("failed to load standard expenditure kind: %v", err)
	}
	computerKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "computer",
	})
	if err != nil {
		t.Fatalf("failed to load computer expenditure kind: %v", err)
	}
	standardKindID := standardKind.Id
	sponsorshipKindID := sponsorshipKind.Id
	computerKindID := computerKind.Id

	// Helper to convert a JSON body string into multipart/form-data with a tiny PNG attachment
	makeMultipart := func(jsonBody string) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		ensureDefaultPOKind(m)
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "receipt.png")
		if err != nil {
			return nil, "", err
		}
		// Minimal PNG header so mime detection passes (image/png)
		if _, err := fw.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}); err != nil {
			return nil, "", err
		}
		contentType := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, contentType, nil
	}

	var scenarios []tests.ApiScenario
	// otherwise valid purchase order fails when job is Closed
	{
		b, ct, err := makeMultipart(fmt.Sprintf(`{
            "uid": "rzr98oadsp9qc11",
            "date": "2024-09-01",
						"division": "vccd5fo56ctbigh",
            "description": "test purchase order",
            "payment_type": "Expense",
            "total": 1234.56,
            "vendor": "2zqxtsmymf670ha",
            "approver": "etysnrlup2f6bak",
            "status": "Unapproved",
            "type": "One-Time",
            "job": "%s"
        }`, "zke3cs3yipplwtu"))
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails when job is Closed",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"job":{"code":"not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// fails when job division is not allocated for the selected job
	{
		b, ct, err := makeMultipart(`{
            "uid": "rzr98oadsp9qc11",
            "date": "2024-09-01",
            "division": "90drdtwx5v4ew70",
            "description": "job PO with unallocated division",
            "payment_type": "Expense",
            "total": 1234.56,
            "vendor": "2zqxtsmymf670ha",
            "approver": "etysnrlup2f6bak",
            "status": "Unapproved",
            "type": "One-Time",
            "job": "test_job_w_rs"
        }`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails when division is not allocated to selected job",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"division":{"code":"division_not_allowed"`,
				`Division BM is not allocated to this job`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// fails when a job is set and kind does not allow jobs
	{
		b, ct, err := makeMultipart(fmt.Sprintf(`{
            "uid": "rzr98oadsp9qc11",
            "date": "2024-09-01",
						"division": "vccd5fo56ctbigh",
            "description": "job PO with non-standard kind",
            "payment_type": "Expense",
            "total": 1234.56,
            "vendor": "2zqxtsmymf670ha",
            "approver": "etysnrlup2f6bak",
            "status": "Unapproved",
            "type": "One-Time",
            "job": "cjf0kt0defhq480",
            "kind": "%s"
        }`, sponsorshipKindID))
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails when job is set and kind does not allow jobs",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"kind":{"code":"invalid_kind_for_job"`,
				`"message":"selected kind does not allow job"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// valid purchase order can be created with a job and computer kind
	{
		b, ct, err := makeMultipart(fmt.Sprintf(`{
	            "uid": "rzr98oadsp9qc11",
	            "date": "2024-09-01",
						"division": "vccd5fo56ctbigh",
	            "description": "job PO with computer kind",
	            "payment_type": "Expense",
	            "total": 123.45,
	            "vendor": "2zqxtsmymf670ha",
	            "approver": "etysnrlup2f6bak",
	            "status": "Unapproved",
	            "type": "One-Time",
            "job": "cjf0kt0defhq480",
            "kind": "%s"
        }`, computerKindID))
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "valid purchase order is created when job is set and kind is computer",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				fmt.Sprintf(`"kind":"%s"`, computerKindID),
				`"job":"cjf0kt0defhq480"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// valid purchase order can be created without attachment
	{
		b := bytes.NewBufferString(fmt.Sprintf(`{
		            "uid": "rzr98oadsp9qc11",
			            "date": "2024-09-01",
						"division": "vccd5fo56ctbigh",
		            "description": "test purchase order",
		            "payment_type": "Expense",
		            "total": 1234.56,
		            "vendor": "2zqxtsmymf670ha",
		            "approver": "etysnrlup2f6bak",
					"priority_second_approver": "6bq4j0eb26631dy",
		            "status": "Unapproved",
	            "type": "One-Time",
	            "kind": "%s"
	        }`, standardKindID))
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "valid purchase order is created without attachment",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// valid purchase order is created
	{
		b, ct, err := makeMultipart(`{
            "uid": "rzr98oadsp9qc11",
	            "date": "2024-09-01",
						"division": "vccd5fo56ctbigh",
            "description": "test purchase order",
            "payment_type": "Expense",
            "total": 1234.56,
            "vendor": "2zqxtsmymf670ha",
            "approver": "etysnrlup2f6bak",
	            "priority_second_approver": "6bq4j0eb26631dy",
            "status": "Unapproved",
            "type": "One-Time"
        }`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "valid purchase order is created",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	// branch defaults from creator profile when omitted and no job is set
	{
		b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
					"date": "2024-09-01",
						"division": "vccd5fo56ctbigh",
            "description": "default branch assignment",
            "payment_type": "Expense",
            "total": 1234.56,
            "vendor": "2zqxtsmymf670ha",
            "approver": "etysnrlup2f6bak",
            "priority_second_approver": "6bq4j0eb26631dy",
            "status": "Unapproved",
            "type": "One-Time"
        }`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "branch defaults from creator profile when omitted",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"80875lm27v8wgi4"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	// job branch overrides an explicit branch when job is set
	{
		b, ct, err := makeMultipart(`{
            "uid": "rzr98oadsp9qc11",
            "date": "2024-09-01",
	          "division": "vccd5fo56ctbigh",
            "description": "manual branch override",
            "payment_type": "Expense",
            "total": 1234.56,
            "vendor": "2zqxtsmymf670ha",
            "approver": "etysnrlup2f6bak",
            "priority_second_approver": "6bq4j0eb26631dy",
            "status": "Unapproved",
            "type": "One-Time",
            "job": "cjf0kt0defhq480",
            "branch": "xeq9q81q5307f70"
        }`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "job branch overrides explicit branch when job is set",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"job":"cjf0kt0defhq480"`,
				`"branch":"80875lm27v8wgi4"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	// otherwise valid PO with Inactive division fails
	{
		b, ct, err := makeMultipart(`{
	            "uid": "rzr98oadsp9qc11",
	            "date": "2024-09-01",
	            "division": "apkev2ow1zjtm7w",
	            "description": "po with inactive division",
	            "payment_type": "Expense",
	            "total": 1234.56,
	            "vendor": "2zqxtsmymf670ha",
	            "approver": "wegviunlyr2jjjv",
	            "status": "Unapproved",
	            "type": "One-Time"
	        }`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "otherwise valid purchase order with Inactive division fails",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"division":{"code":"not_active"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// recurring purchase order requires end_date and frequency
	{
		b, ct, err := makeMultipart(`{
            "uid": "rzr98oadsp9qc11",
            "date": "2024-09-01",
            "division": "vccd5fo56ctbigh",
            "description": "test purchase order",
            "payment_type": "Expense",
            "total": 1234.56,
            "vendor": "2zqxtsmymf670ha",
            "approver": "etysnrlup2f6bak",
            "priority_second_approver": "6bq4j0eb26631dy",
            "status": "Unapproved",
            "type": "Recurring",
            "end_date": "2024-11-01",
            "frequency": "Monthly"
        }`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "recurring purchase order requires end_date and frequency",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	// recurring purchase order fails without end_date
	{
		b, ct, err := makeMultipart(`{
            "uid": "rzr98oadsp9qc11",
            "date": "2024-09-01",
            "division": "vccd5fo56ctbigh",
            "description": "test purchase order",
            "payment_type": "Expense",
            "total": 1234.56,
            "vendor": "2zqxtsmymf670ha",
            "approver": "etysnrlup2f6bak",
            "status": "Unapproved",
            "type": "Recurring",
            "frequency": "Monthly"
        }`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "recurring purchase order fails without end_date",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"end_date":{"code":"value_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "Recurring",
			"end_date": "2024-11-01"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "recurring purchase fails without frequency",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"frequency":{"code":"value_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "Recurring",
			"end_date": "2024-10-01",
			"frequency": "Monthly"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "recurring purchase order fails with less than 2 occurrences",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"global":{"code":"fewer_than_two_occurrences"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "Recurring",
			"end_date": "2024-09-01",
			"frequency": "Monthly"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "recurring purchase order fails if end_date is not after start_date",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"end_date":{"code":"end_date_not_after_start_date"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"priority_second_approver": "66ct66w380ob6w8",
				"status": "Unapproved",
				"type": "Recurring",
				"end_date": "2024-11-01",
				"frequency": "Weekly"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "recurring purchase order allows other frequencies",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "Recurring",
			"end_date": "2024-11-01",
			"frequency": "Invalid"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "recurring purchase order fails when frequency is not valid",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"frequency":{"code":"invalid_frequency"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "tqqf7q0f3378rvp",
			"status": "Unapproved",
			"type": "One-Time"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "otherwise valid purchase order fails when approver is set non-qualified user",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"approver":{"code":"invalid_approver_for_stage"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "",
			"status": "Unapproved",
			"type": "One-Time"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "otherwise valid purchase order fails when approver is set to blank string or missing",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"approver":{"code":"invalid_approver_for_stage"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"parent_po": "ly8xyzpuj79upq1",
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "this one is cumulative",
			"payment_type": "OnAccount",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"priority_second_approver": "6bq4j0eb26631dy",
			"status": "Unapproved",
			"type": "One-Time",
			"job": "cjf0kt0defhq480",
			"category": "t5nmdl188gtlhz0"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "valid child purchase order is created",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	// We need a test child PO that is status Active
	{
		b, ct, err := makeMultipart(`{
			"parent_po": "25046ft47x49cc2",
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "this one is cumulative",
			"payment_type": "OnAccount",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "One-Time",
			"job": "cjf0kt0defhq480",
			"category": "t5nmdl188gtlhz0"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "a child purchase order cannot itself be a parent",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"parent_po":{"code":"child_po_cannot_be_parent"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"parent_po": "ly8xyzpuj79upq1",
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "this one is cumulative",
			"payment_type": "OnAccount",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "Cumulative",
			"job": "cjf0kt0defhq480",
			"category": "t5nmdl188gtlhz0"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "child purchase order may not be of type Cumulative",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"type":{"code":"validation_in_invalid"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"parent_po": "ly8xyzpuj79upq1",
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "this one is cumulative",
			"payment_type": "OnAccount",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "Recurring",
			"end_date": "2024-11-01",
			"frequency": "Monthly",
			"job": "cjf0kt0defhq480",
			"category": "t5nmdl188gtlhz0"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "child purchase order may not be of type Recurring",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"type":{"code":"validation_in_invalid"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"parent_po": "y660i6a14ql2355",
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "this one is cumulative",
			"payment_type": "OnAccount",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "One-Time",
			"job": "cjf0kt0defhq480",
			"category": "t5nmdl188gtlhz0"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails when other child POs with status 'Unapproved' exist",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"parent_po":{"code":"existing_children_with_blocking_status"`,
			},
			ExpectedEvents: map[string]int{
				"*":                     0,
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"parent_po": "2plsetqdxht7esg",
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "this one is cumulative",
			"payment_type": "OnAccount",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "One-Time",
			"job": "cjf0kt0defhq480",
			"category": "t5nmdl188gtlhz0"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails when parent_po is not cumulative",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"parent_po":{"code":"invalid_type"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"parent_po": "ly8xyzpuj79upq1",
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "this one is cumulative",
			"payment_type": "OnAccount",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "One-Time",
			"job": "tt4eipt6wapu9zh",
			"category": "he1f7oej613mxh7"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails when job of child purchase order does not match job of parent purchase order",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"job":{"code":"value_mismatch"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 1234.56,
			"vendor": "ctswqva5onxj75q",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "One-Time"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "otherwise valid purchase order with Inactive vendor fails",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"status":400`,
				`"message":"hook error when validating vendor"`,
				`"vendor":{"code":"inactive_vendor"`,
				`"message":"provided vendor is not active"`,
			},
			ExpectedEvents: map[string]int{},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	/*
	   This test verifies the basic auto-approval flow for purchase orders.
	   When a user with the po_approver claim (empty divsions property of po_approver_props = all divisions) creates a PO:
	   1. The PO should be auto-approved immediately:
	      - approved timestamp should be set to current date/time
	      - approver should be set to the creator's ID
	   2. Status should become "Active" (since no second approval needed for low value)
	   3. PO number should be generated (format: YYYY-NNNN)

	   Test setup:
	   - Uses user wegviunlyr2jjjv (fakemanager@fakesite.xyz) who has po_approver claim
	   - Sets PO total to random value below tier1 to avoid triggering second approval
	   - Uses correct auth token matching the creator's ID

	   Verification points:
	   - approved: Checks timestamp starts with current date
	   - status: Must be "Active"
	   - po_number: Must start with current year
	   - approver: Must be creator's ID (wegviunlyr2jjjv)
	*/
	{
		json := fmt.Sprintf(`{
			"uid": "wegviunlyr2jjjv",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": %.2f,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "One-Time"
		}`, rand.Float64()*(tier1-1.0)+1.0)
		b, ct, err := makeMultipart(json)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "purchase order is not automatically approved when creator has po_approver claim",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": poApproverToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"status":"Unapproved"`,
				`"po_number":""`,
				`"approver":"etysnrlup2f6bak"`, // Original approver remains
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	/*
	   These tests verify division-specific auto-approval for purchase orders.
	   User fatt@mac.com (id: etysnrlup2f6bak) has po_approver claim with divisions property:
	   ["hcd86z57zjty6jo", "fy4i9poneukvq9u", "vccd5fo56ctbigh"] on the po_approver_props record

	   Test 1 (Success case):
	   - Creates PO with division "vccd5fo56ctbigh" (in user's po_approver_props divisions property)
	   - Should auto-approve since user has permission for this division
	   - Verifies: approval timestamp, Active status, PO number generation
	   - Creator becomes approver

	   Test 2 (Failure case):
	   - Creates PO with division "ngpjzurmkrfl8fo" (not in user's po_approver_props divisions property)
	   - Uses wegviunlyr2jjjv as approver (has empty po_approver_props divisions property = all divisions)
	   - Should succeed (200) but not auto-approve
	   - Verifies: no approval, Unapproved status, original approver remains

	   Both tests:
	   - Use random total below tier1 to avoid second approval
	   - Use correct auth token for fatt@mac.com
	   - Match uid to authenticated user's ID
	*/
	{
		json := fmt.Sprintf(`{
			"uid": "etysnrlup2f6bak",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": %.2f,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "One-Time"
		}`, rand.Float64()*(tier1-1.0)+1.0)
		b, ct, err := makeMultipart(json)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "purchase order is not automatically approved when creator has po_approver claim for non-matching division",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": divisionApproverToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"status":"Unapproved"`,
				`"po_number":""`,
				`"approver":"etysnrlup2f6bak"`, // Original approver remains
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		json := fmt.Sprintf(`{
			"uid": "etysnrlup2f6bak",
			"date": "2024-09-01",
			"division": "ngpjzurmkrfl8fo",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": %.2f,
			"vendor": "2zqxtsmymf670ha",
			"approver": "wegviunlyr2jjjv",
			"status": "Unapproved",
			"type": "One-Time"
		}`, rand.Float64()*(tier1-1.0)+1.0)
		b, ct, err := makeMultipart(json)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "purchase order is not auto-approved when creator has po_approver claim but non-matching division",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": divisionApproverToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,                // Should not be approved
				`"status":"Unapproved"`,        // Status should remain Unapproved
				`"approver":"wegviunlyr2jjjv"`, // Original approver should remain
				`"po_number":""`,               // No PO number should be generated
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	/*
	   This test verifies auto-approval of high-value purchase orders by users with elevated claims.
	   User hal@2005.com (id: 66ct66w380ob6w8) has:
	   - po_approver claim with empty divisions property on the po_approver_props record (can approve for any division)
	   - po_approver_tier3 claim (can provide second approval for high-value POs)

	   Test verifies that when this user creates a high-value PO:
	   1. First approval is automatic (due to po_approver claim)
	   2. Second approval is also automatic (due to po_approver_tier3 claim)
	   3. Status becomes Active and PO number is generated
	   4. Creator is set as both approver and second_approver

	   The test:
	   - Uses total above tier2 to trigger second approval requirement
	   - Uses random division (since user has empty po_approver_props divisions property)
	   - Verifies all approval fields and timestamps
	*/
	{
		json := fmt.Sprintf(`{
			"uid": "66ct66w380ob6w8",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": %.2f,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"priority_second_approver": "66ct66w380ob6w8",
			"status": "Unapproved",
			"type": "One-Time"
		}`, rand.Float64()*(1000.0)+tier2)
		b, ct, err := makeMultipart(json)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "purchase order is not automatically approved when creator has po_approver and po_approver_tier3 claims",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": po_approver_tier3Token, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"second_approval":""`,
				`"status":"Unapproved"`,
				`"po_number":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	/*
	   This test verifies auto-approval of mid-range value purchase orders by users with po_approver_tier2 claim.
	   User author@soup.com (id: f2j5a8vk006baub) has:
	   - po_approver claim with empty divisions property on the po_approver_props record (can approve for any division)
	   - po_approver_tier2 claim (can provide second approval for POs between tier1 and tier2)

	   Test verifies that when this user creates a mid-range value PO:
	   1. First approval is automatic (due to po_approver claim)
	   2. Second approval is also automatic (due to po_approver_tier2 claim)
	   3. Status becomes Active and PO number is generated
	   4. Creator is set as both approver and second_approver

	   The test:
	   - Uses total between tier1 and tier2
	   - Uses random division (since user has empty po_approver_props divisions property)
	   - Verifies all approval fields and timestamps
	*/
	{
		json := fmt.Sprintf(`{
			"uid": "f2j5a8vk006baub",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": %.2f,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"priority_second_approver": "6bq4j0eb26631dy",
			"status": "Unapproved",
			"type": "One-Time"
		}`, rand.Float64()*(tier2-tier1)+tier1)
		b, ct, err := makeMultipart(json)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "purchase order is not automatically approved when creator has po_approver and po_approver_tier2 claims",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": po_approver_tier2Token, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"second_approval":""`,
				`"status":"Unapproved"`,
				`"po_number":""`,
				`"approver":"etysnrlup2f6bak"`, // Original approver remains
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		// Dual-required self-bypass setup is allowed when creator is second-stage qualified.
		json := fmt.Sprintf(`{
			"uid": "66ct66w380ob6w8",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "dual required self bypass setup",
			"payment_type": "Expense",
			"total": %.2f,
			"vendor": "2zqxtsmymf670ha",
			"approver": "66ct66w380ob6w8",
			"priority_second_approver": "66ct66w380ob6w8",
			"status": "Unapproved",
			"type": "One-Time",
			"kind": "%s"
		}`, tier1+100, standardKindID)
		b, ct, err := makeMultipart(json)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "dual-required save allows self assignment for creator who is second-stage qualified",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": po_approver_tier3Token, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approver":"66ct66w380ob6w8"`,
				`"priority_second_approver":"66ct66w380ob6w8"`,
				`"approved":""`,
				`"second_approval":""`,
				`"status":"Unapproved"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the approval-required notification
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	{
		// Guard: self assignment is invalid when creator is not second-stage qualified.
		json := fmt.Sprintf(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "dual required self bypass invalid",
			"payment_type": "Expense",
			"total": %.2f,
			"vendor": "2zqxtsmymf670ha",
			"approver": "rzr98oadsp9qc11",
			"priority_second_approver": "rzr98oadsp9qc11",
			"status": "Unapproved",
			"type": "One-Time",
			"kind": "%s"
		}`, tier1+100, standardKindID)
		b, ct, err := makeMultipart(json)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "dual-required save rejects self assignment for creator who is not second-stage qualified",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"approver":{"code":"invalid_approver_for_stage"`,
				`"priority_second_approver":{"code":"invalid_priority_second_approver_for_stage"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	// Test that priority_second_approver must be an authorized approver for the PO amount
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "One-Time",
			"priority_second_approver": "u_no_claims"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails when priority_second_approver is not authorized for the PO amount",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"priority_second_approver":{"code":"invalid_priority_second_approver_for_stage"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	// Test that priority_second_approver must be an active user
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 1234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "One-Time",
			"priority_second_approver": "u_inactive"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails when priority_second_approver is inactive",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"priority_second_approver":{"code":"priority_second_approver_not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}
	// Test that approver must be an active user
	{
		b, ct, err := makeMultipart(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order with inactive approver",
			"payment_type": "Expense",
			"total": 99,
			"vendor": "2zqxtsmymf670ha",
			"approver": "u_inactive",
			"status": "Unapproved",
			"type": "One-Time"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails when approver is inactive",
			Method:         http.MethodPost,
			URL:            "/api/collections/purchase_orders/records",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"approver":{"code":"approver_not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestPurchaseOrdersCreate_DuplicateAttachmentFails verifies that creating a purchase order
// with an attachment that has the same hash as an existing purchase order fails.
func TestPurchaseOrdersCreate_DuplicateAttachmentFails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	// Helper to create multipart form data with a specific file content
	makeMultipartWithContent := func(jsonBody string, fileContent []byte) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		ensureDefaultPOKind(m)
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "receipt.png")
		if err != nil {
			return nil, "", err
		}
		if _, err := fw.Write(fileContent); err != nil {
			return nil, "", err
		}
		contentType := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, contentType, nil
	}

	// Use a specific file content that will produce a consistent hash
	duplicateFileContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xAB, 0xCD, 0xEF, 0x01}

	scenario := tests.ApiScenario{
		Name:           "duplicate attachment fails with field-level error",
		Method:         http.MethodPost,
		URL:            "/api/collections/purchase_orders/records",
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"attachment":{"code":"duplicate_file"`,
			`"message":"This file has already been uploaded to another purchase order"`,
		},
		ExpectedEvents: map[string]int{
			"OnRecordCreateRequest": 1,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	// Create the request body for the duplicate attempt
	b, ct, err := makeMultipartWithContent(`{
		"uid": "rzr98oadsp9qc11",
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "second PO with same attachment",
		"payment_type": "Expense",
		"total": 600,
		"vendor": "2zqxtsmymf670ha",
		"approver": "etysnrlup2f6bak",
		"status": "Unapproved",
		"type": "One-Time"
	}`, duplicateFileContent)
	if err != nil {
		t.Fatal(err)
	}
	scenario.Body = b
	scenario.Headers = map[string]string{"Authorization": recordToken, "Content-Type": ct}

	scenario.Test(t)
}

// TestPurchaseOrdersUpdate_DuplicateAttachmentFails verifies that updating a purchase order
// with an attachment that has the same hash as another purchase order fails.
func TestPurchaseOrdersUpdate_DuplicateAttachmentFails(t *testing.T) {
	// Use author@soup.com who can update PO gal6e5la2fa4rpn (Unapproved status)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Helper to create multipart form data with a specific file content
	makeMultipartWithContent := func(jsonBody string, fileContent []byte) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		ensureDefaultPOKind(m)
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "receipt.png")
		if err != nil {
			return nil, "", err
		}
		if _, err := fw.Write(fileContent); err != nil {
			return nil, "", err
		}
		contentType := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, contentType, nil
	}

	// Use a specific file content that will produce a consistent hash
	duplicateFileContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xFE, 0xDC, 0xBA, 0x98}

	scenario := tests.ApiScenario{
		Name:           "updating PO with duplicate attachment fails",
		Method:         http.MethodPatch,
		URL:            "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"attachment":{"code":"duplicate_file"`,
			`"message":"This file has already been uploaded to another purchase order"`,
		},
		ExpectedEvents: map[string]int{
			"OnRecordUpdateRequest": 1,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	// Create the request body for the update with duplicate attachment
	b, ct, err := makeMultipartWithContent(`{
		"uid": "f2j5a8vk006baub",
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "trying to update with duplicate attachment",
		"payment_type": "Expense",
		"total": 2234.56,
		"vendor": "2zqxtsmymf670ha",
		"approver": "etysnrlup2f6bak",
		"status": "Unapproved",
		"type": "Cumulative"
	}`, duplicateFileContent)
	if err != nil {
		t.Fatal(err)
	}
	scenario.Body = b
	scenario.Headers = map[string]string{"Authorization": recordToken, "Content-Type": ct}

	scenario.Test(t)
}

func TestPurchaseOrdersUpdate(t *testing.T) {

	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	app := testutils.SetupTestApp(t)
	sponsorshipKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "sponsorship",
	})
	if err != nil {
		t.Fatalf("failed to load sponsorship expenditure kind: %v", err)
	}
	standardKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "standard",
	})
	if err != nil {
		t.Fatalf("failed to load standard expenditure kind: %v", err)
	}
	computerKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "computer",
	})
	if err != nil {
		t.Fatalf("failed to load computer expenditure kind: %v", err)
	}
	standardKindID := standardKind.Id
	sponsorshipKindID := sponsorshipKind.Id
	computerKindID := computerKind.Id

	// multipart builder for updates with attachment
	updateMultipart := func(jsonBody string) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		ensureDefaultPOKind(m)
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "update.png")
		if err != nil {
			return nil, "", err
		}
		if _, err := fw.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}); err != nil {
			return nil, "", err
		}
		ct := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, ct, nil
	}

	var scenarios []tests.ApiScenario

	// valid purchase order is updated
	{
		b, ct, err := updateMultipart(`{
			"uid": "f2j5a8vk006baub",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 2234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"priority_second_approver": "6bq4j0eb26631dy",
			"status": "Unapproved",
			"type": "Cumulative"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "valid purchase order is updated",
			Method:         http.MethodPatch,
			URL:            "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// valid purchase order update can clear attachment without uploading a new file
	{
		b := bytes.NewBufferString(fmt.Sprintf(`{
			"uid": "f2j5a8vk006baub",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 2234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"priority_second_approver": "6bq4j0eb26631dy",
			"status": "Unapproved",
			"type": "Cumulative",
			"kind": "%s",
			"attachment": ""
		}`, standardKindID))
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "valid purchase order update clears attachment without new upload",
			Method:         http.MethodPatch,
			URL:            "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"attachment":""`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// otherwise valid purchase order with Inactive vendor fails
	{
		b, ct, err := updateMultipart(`{
			"uid": "f2j5a8vk006baub",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 2234.56,
			"vendor": "ctswqva5onxj75q",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "Cumulative"
		}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "otherwise valid purchase order with Inactive vendor fails",
			Method:         http.MethodPatch,
			URL:            "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"status":400`,
				`"message":"hook error when validating vendor"`,
				`"vendor":{"code":"inactive_vendor"`,
				`"message":"provided vendor is not active"`,
			},
			ExpectedEvents: map[string]int{},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// valid purchase order updates can keep job with computer kind
	{
		b, ct, err := updateMultipart(fmt.Sprintf(`{
				"uid": "f2j5a8vk006baub",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 223.45,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Cumulative",
			"job": "cjf0kt0defhq480",
			"kind": "%s"
		}`, computerKindID))
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "valid purchase order updates when job is set and kind is computer",
			Method:         http.MethodPatch,
			URL:            "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				fmt.Sprintf(`"kind":"%s"`, computerKindID),
				`"job":"cjf0kt0defhq480"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// updates cannot set status to Active directly
	{
		b, ct, err := updateMultipart(`{
				"uid": "f2j5a8vk006baub",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "attempt direct active transition",
				"payment_type": "Expense",
				"total": 2234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Active",
				"type": "Cumulative"
			}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails to update purchase order status to Active directly",
			Method:         http.MethodPatch,
			URL:            "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"hook error when validating status"`,
				`"status":{"code":"invalid_status"`,
				`"status must be Unapproved when creating or updating purchase orders"`,
			},
			ExpectedEvents: map[string]int{},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// updates cannot set status to Cancelled directly
	{
		b, ct, err := updateMultipart(`{
				"uid": "f2j5a8vk006baub",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "attempt direct cancelled transition",
				"payment_type": "Expense",
				"total": 2234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Cancelled",
				"type": "Cumulative"
			}`)
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails to update purchase order status to Cancelled directly",
			Method:         http.MethodPatch,
			URL:            "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"hook error when validating status"`,
				`"status":{"code":"invalid_status"`,
				`"status must be Unapproved when creating or updating purchase orders"`,
			},
			ExpectedEvents: map[string]int{},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	// fails to update when a job is set and kind does not allow jobs
	{
		b, ct, err := updateMultipart(fmt.Sprintf(`{
			"uid": "f2j5a8vk006baub",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "test purchase order",
			"payment_type": "Expense",
			"total": 2234.56,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"status": "Unapproved",
			"type": "Cumulative",
			"job": "cjf0kt0defhq480",
			"kind": "%s"
		}`, sponsorshipKindID))
		if err != nil {
			t.Fatal(err)
		}
		scenarios = append(scenarios, tests.ApiScenario{
			Name:           "fails to update when job is set and kind does not allow jobs",
			Method:         http.MethodPatch,
			URL:            "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
			Body:           b,
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"kind":{"code":"invalid_kind_for_job"`,
				`"message":"selected kind does not allow job"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		})
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPurchaseOrdersUpdate_FirstApprovedEditBehavior(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const firstApprovedPOID = "2blv18f40i2q373"
	const firstApprovedApproverChangePOID = "po1stgready0001"
	const draftPOID = "gal6e5la2fa4rpn"

	app := testutils.SetupTestApp(t)
	approverChangePO, err := app.FindRecordById("purchase_orders", firstApprovedApproverChangePOID)
	if err != nil {
		t.Fatalf("failed loading first-approved approver-change fixture: %v", err)
	}
	approverChangePolicy, err := utilities.GetPOApproverPolicy(
		app,
		approverChangePO.GetString("division"),
		approverChangePO.GetFloat("approval_total"),
		approverChangePO.GetString("kind"),
		approverChangePO.GetString("job") != "",
	)
	if err != nil {
		t.Fatalf("failed computing approver policy for fixture: %v", err)
	}
	approverChangeTarget := ""
	currentApprover := approverChangePO.GetString("approver")
	for _, candidate := range approverChangePolicy.FirstStageApprovers {
		if candidate.ID != "" && candidate.ID != currentApprover {
			approverChangeTarget = candidate.ID
			break
		}
	}
	if approverChangeTarget == "" {
		t.Skip("no alternate first-stage approver available for approver-change fixture")
	}

	var meaningfulBeforeNotificationCount int
	var approverChangeBeforeNotificationCount int
	var noOpBeforeNotificationCount int
	var draftBeforeNotificationCount int

	scenarios := []tests.ApiScenario{
		{
			Name:   "approver change on first-approved unapproved PO resets approvals and re-notifies",
			Method: http.MethodPatch,
			URL:    "/api/collections/purchase_orders/records/" + firstApprovedApproverChangePOID,
			Body: bytes.NewBufferString(fmt.Sprintf(`{
					"uid": "f2j5a8vk006baub",
					"date": "2024-01-31",
					"division": "fy4i9poneukvq9u",
					"description": "single-stage approved-not-active fixture",
					"payment_type": "OnAccount",
				"total": 329.01,
				"vendor": "2zqxtsmymf670ha",
				"approver": "%s",
				"status": "Unapproved",
				"type": "One-Time",
				"job": "u09fwwcg07y03m7",
				"category": "",
				"kind": "l3vtlbqg529m52j"
			}`, approverChangeTarget)),
			Headers: map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				var row struct {
					Count int `db:"count"`
				}
				if err := app.DB().NewQuery(`
					SELECT COUNT(*) AS count
					FROM notifications
					WHERE json_extract(data, '$.POId') = {:poID}
				`).Bind(dbx.Params{"poID": firstApprovedApproverChangePOID}).One(&row); err != nil {
					tb.Fatalf("failed counting baseline notifications for approver-change PO: %v", err)
				}
				approverChangeBeforeNotificationCount = row.Count
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
				var row struct {
					Count int `db:"count"`
				}
				if err := app.DB().NewQuery(`
					SELECT COUNT(*) AS count
					FROM notifications
					WHERE json_extract(data, '$.POId') = {:poID}
				`).Bind(dbx.Params{"poID": firstApprovedApproverChangePOID}).One(&row); err != nil {
					tb.Fatalf("failed counting notifications after approver-change update: %v", err)
				}
				if row.Count != approverChangeBeforeNotificationCount+1 {
					tb.Fatalf("expected notification count to increase by 1 for approver change, got before=%d after=%d", approverChangeBeforeNotificationCount, row.Count)
				}
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"second_approval":""`,
				fmt.Sprintf(`"approver":"%s"`, approverChangeTarget),
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
				"OnRecordCreate": 1, // new notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "meaningful edit on first-approved unapproved PO resets approvals and re-notifies",
			Method: http.MethodPatch,
			URL:    "/api/collections/purchase_orders/records/" + firstApprovedPOID,
			Body: bytes.NewBufferString(`{
						"uid": "f2j5a8vk006baub",
						"date": "2025-01-29",
						"division": "vccd5fo56ctbigh",
					"description": "Higher-value unapproved PO that already has first approval (edited)",
					"payment_type": "OnAccount",
				"total": 1022.69,
				"vendor": "mmgxrnn144767x7",
				"approver": "wegviunlyr2jjjv",
				"priority_second_approver": "6bq4j0eb26631dy",
				"status": "Unapproved",
				"type": "One-Time",
				"job": "u09fwwcg07y03m7",
				"category": "",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				var row struct {
					Count int `db:"count"`
				}
				if err := app.DB().NewQuery(`
					SELECT COUNT(*) AS count
					FROM notifications
					WHERE json_extract(data, '$.POId') = {:poID}
				`).Bind(dbx.Params{"poID": firstApprovedPOID}).One(&row); err != nil {
					tb.Fatalf("failed counting baseline notifications: %v", err)
				}
				meaningfulBeforeNotificationCount = row.Count
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
				var row struct {
					Count int `db:"count"`
				}
				if err := app.DB().NewQuery(`
					SELECT COUNT(*) AS count
					FROM notifications
					WHERE json_extract(data, '$.POId') = {:poID}
				`).Bind(dbx.Params{"poID": firstApprovedPOID}).One(&row); err != nil {
					tb.Fatalf("failed counting notifications after update: %v", err)
				}
				if row.Count != meaningfulBeforeNotificationCount+1 {
					tb.Fatalf("expected notification count to increase by 1, got before=%d after=%d", meaningfulBeforeNotificationCount, row.Count)
				}
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"second_approval":""`,
				`"description":"Higher-value unapproved PO that already has first approval (edited)"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
				"OnRecordCreate": 1, // new notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "no-op update on first-approved unapproved PO preserves approvals and does not re-notify",
			Method: http.MethodPatch,
			URL:    "/api/collections/purchase_orders/records/" + firstApprovedPOID,
			Body: bytes.NewBufferString(`{
						"uid": "f2j5a8vk006baub",
						"date": "2025-01-29",
						"division": "vccd5fo56ctbigh",
						"description": "Higher-value unapproved PO that already has first approval ",
						"payment_type": "OnAccount",
				"total": 1022.69,
				"vendor": "mmgxrnn144767x7",
				"approver": "wegviunlyr2jjjv",
				"priority_second_approver": "6bq4j0eb26631dy",
				"status": "Unapproved",
				"type": "One-Time",
				"job": "u09fwwcg07y03m7",
				"category": "",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				var row struct {
					Count int `db:"count"`
				}
				if err := app.DB().NewQuery(`
					SELECT COUNT(*) AS count
					FROM notifications
					WHERE json_extract(data, '$.POId') = {:poID}
				`).Bind(dbx.Params{"poID": firstApprovedPOID}).One(&row); err != nil {
					tb.Fatalf("failed counting baseline notifications: %v", err)
				}
				noOpBeforeNotificationCount = row.Count
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
				var row struct {
					Count int `db:"count"`
				}
				if err := app.DB().NewQuery(`
					SELECT COUNT(*) AS count
					FROM notifications
					WHERE json_extract(data, '$.POId') = {:poID}
				`).Bind(dbx.Params{"poID": firstApprovedPOID}).One(&row); err != nil {
					tb.Fatalf("failed counting notifications after no-op update: %v", err)
				}
				if row.Count != noOpBeforeNotificationCount {
					tb.Fatalf("expected notification count to remain unchanged, got before=%d after=%d", noOpBeforeNotificationCount, row.Count)
				}
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":"2025-01-29 14:22:29.563Z"`,
				`"second_approval":""`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "fully approved active PO remains non-editable",
			Method: http.MethodPatch,
			URL:    "/api/collections/purchase_orders/records/d8463q483f3da28",
			Body: bytes.NewBufferString(`{
				"uid": "f2j5a8vk006baub",
				"description": "attempted edit on active PO"
			}`),
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "meaningful edit on draft PO does not re-notify approver",
			Method: http.MethodPatch,
			URL:    "/api/collections/purchase_orders/records/" + draftPOID,
			Body: bytes.NewBufferString(`{
				"uid": "f2j5a8vk006baub",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "Unapproved PO for Dirt slugger (edited draft)",
				"payment_type": "OnAccount",
				"total": 329.01,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "One-Time",
				"job": "",
				"category": "",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				var row struct {
					Count int `db:"count"`
				}
				if err := app.DB().NewQuery(`
					SELECT COUNT(*) AS count
					FROM notifications
					WHERE json_extract(data, '$.POId') = {:poID}
				`).Bind(dbx.Params{"poID": draftPOID}).One(&row); err != nil {
					tb.Fatalf("failed counting baseline notifications for draft PO: %v", err)
				}
				draftBeforeNotificationCount = row.Count
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
				var row struct {
					Count int `db:"count"`
				}
				if err := app.DB().NewQuery(`
					SELECT COUNT(*) AS count
					FROM notifications
					WHERE json_extract(data, '$.POId') = {:poID}
				`).Bind(dbx.Params{"poID": draftPOID}).One(&row); err != nil {
					tb.Fatalf("failed counting notifications after draft update: %v", err)
				}
				if row.Count != draftBeforeNotificationCount {
					tb.Fatalf("expected notification count to remain unchanged for draft edit, got before=%d after=%d", draftBeforeNotificationCount, row.Count)
				}
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"description":"Unapproved PO for Dirt slugger (edited draft)"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPurchaseOrdersUpdate_CannotModifyPONumberForNonUnapprovedStatuses(t *testing.T) {
	authorToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	noclaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name              string
		token             string
		poID              string
		attemptedPONumber string
		expectedPONumber  string
	}{
		{
			name:              "cannot modify po_number on active purchase order",
			token:             authorToken,
			poID:              "y660i6a14ql2355",
			attemptedPONumber: "2599-9999",
			expectedPONumber:  "2025-0001",
		},
		{
			name:              "cannot modify po_number on cancelled purchase order",
			token:             authorToken,
			poID:              "338568325487lo2",
			attemptedPONumber: "2599-9998",
			expectedPONumber:  "2025-0002",
		},
		{
			name:              "cannot modify po_number on closed purchase order",
			token:             noclaimsToken,
			poID:              "0pia83nnprdlzf8",
			attemptedPONumber: "2599-9997",
			expectedPONumber:  "2025-0006",
		},
	}

	for _, tc := range cases {
		scenario := tests.ApiScenario{
			Name:   tc.name,
			Method: http.MethodPatch,
			URL:    "/api/collections/purchase_orders/records/" + tc.poID,
			Body:   bytes.NewBufferString(fmt.Sprintf(`{"po_number":"%s"}`, tc.attemptedPONumber)),
			Headers: map[string]string{
				"Authorization": tc.token,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
				po, findErr := app.FindRecordById("purchase_orders", tc.poID)
				if findErr != nil {
					tb.Fatalf("failed loading purchase order %s: %v", tc.poID, findErr)
				}
				if po.GetString("po_number") != tc.expectedPONumber {
					tb.Fatalf("po_number changed unexpectedly for %s: expected %s, got %s", tc.poID, tc.expectedPONumber, po.GetString("po_number"))
				}
			},
			TestAppFactory: testutils.SetupTestApp,
		}
		scenario.Test(t)
	}
}

func TestPurchaseOrdersDelete(t *testing.T) {
	/*
		recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
		if err != nil {
			t.Fatal(err)
		}

		nonCreatorToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
		if err != nil {
			t.Fatal(err)
		}
	*/

	scenarios := []tests.ApiScenario{
		// TODO: Add test scenarios for purchase order deletion
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPurchaseOrdersRead(t *testing.T) {
	/*
		creatorToken, err := testutils.GenerateRecordToken("users", "time@test.com")
		if err != nil {
			t.Fatal(err)
		}

		approverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
		if err != nil {
			t.Fatal(err)
		}

		reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
		if err != nil {
			t.Fatal(err)
		}
	*/

	scenarios := []tests.ApiScenario{
		// TODO: Add test scenarios for purchase order reading
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestGeneratePONumber(t *testing.T) {
	currentYear := time.Now().Year() % 100
	currentMonth := time.Now().Month()
	currentPoPrefix := fmt.Sprintf("%d%02d-", currentYear, currentMonth)
	app := testutils.SetupTestApp(t)
	poCollection, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		year          int // 0 for current year
		month         int // 0 for current month
		record        *core.Record
		setup         func(t *testing.T, app *tests.TestApp)
		cleanup       func(t *testing.T, app *tests.TestApp)
		expected      string
		expectedError string
	}{
		{
			// Test Case 1: First Child PO
			// This test verifies that when creating the first child PO for an existing parent:
			// - The parent PO (2024-0008) is found in the database
			// - No existing child POs are found
			// - The generated number follows the format: parent_number + "-01"
			// - The number is unique in the database
			name: "first child PO for 2024-0008",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "2plsetqdxht7esg") // ID of PO 2024-0008
				return r
			}(),
			expected: "2024-0008-01",
		},
		{
			// Test Case 2: Second Child PO
			// This test verifies that when creating a second child PO:
			// - The parent PO (2024-0008) is found
			// - An existing child PO (2024-0008-01) is found
			// - The next sequential number (-02) is generated
			// - The number is unique in the database
			//
			// The test:
			// 1. Sets up by creating the first child PO (2024-0008-01)
			// 2. Attempts to create a second child PO
			// 3. Cleans up by removing all child POs created during the test
			name: "second child PO for 2024-0008",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "2plsetqdxht7esg")
				r.Set("uid", "f2j5a8vk006baub")
				r.Set("type", "One-Time")
				r.Set("date", "2024-01-01")
				r.Set("division", "ngpjzurmkrfl8fo")
				r.Set("description", "Test description")
				r.Set("total", 100.0)
				r.Set("payment_type", "OnAccount")
				r.Set("vendor", "2zqxtsmymf670ha")
				r.Set("approver", "wegviunlyr2jjjv")
				r.Set("status", "Unapproved")
				return r
			}(),
			setup: func(t *testing.T, app *tests.TestApp) {
				// Create and save the first child PO
				firstChild := core.NewRecord(poCollection)
				firstChild.Set("parent_po", "2plsetqdxht7esg")
				firstChild.Set("uid", "f2j5a8vk006baub")
				firstChild.Set("type", "One-Time")
				firstChild.Set("po_number", "2024-0008-01")
				firstChild.Set("date", "2024-01-01")
				firstChild.Set("division", "ngpjzurmkrfl8fo")
				firstChild.Set("description", "Test description")
				firstChild.Set("total", 100.0)
				firstChild.Set("approval_total", 100.0)
				firstChild.Set("payment_type", "OnAccount")
				firstChild.Set("vendor", "2zqxtsmymf670ha")
				firstChild.Set("approver", "wegviunlyr2jjjv")
				firstChild.Set("status", "Unapproved")
				firstChild.Set("kind", utilities.DefaultExpenditureKindID())
				if err := app.Save(firstChild); err != nil {
					t.Fatalf("failed to save first child PO: %v", err)
				}
			},
			cleanup: func(t *testing.T, app *tests.TestApp) {
				// Delete any child POs we created
				records, err := app.FindRecordsByFilter(
					"purchase_orders",
					"parent_po = {:parentId}",
					"",
					0,
					0,
					dbx.Params{"parentId": "2plsetqdxht7esg"},
				)
				if err != nil {
					t.Fatalf("failed to find child POs to clean up: %v", err)
				}
				for _, record := range records {
					if err := app.Delete(record); err != nil {
						t.Fatalf("failed to delete child PO: %v", err)
					}
				}
			},
			expected: "2024-0008-02",
		},
		{
			// Test Case 3: Parent PO Number Generation (for current YYMM)
			// This test verifies that when creating a new parent PO for the current year/month:
			// - A PO is set up with number YYMM-NNNN.
			// - The next generated sequential number is YYMM-(NNNN+1).
			// - The number is unique in the database.
			name:  "sequential parent PO",
			year:  2024,
			month: 1,
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			expected: "2401-0010", // Next after 2024-0009 in test DB
		},
		{
			// Test Case 4: Parent PO Without Number
			// This test verifies that when creating a child PO:
			// - If the parent PO exists but has no PO number
			// - An appropriate error is returned
			name: "parent PO without number",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "gal6e5la2fa4rpn") // ID of a PO without number
				return r
			}(),
			expectedError: "parent PO does not have a PO number",
		},
		{
			// Test Case 5: Maximum Child POs Reached
			// This test verifies that when creating a child PO:
			// - If 99 child POs already exist for the parent
			// - An appropriate error is returned
			//
			// The test:
			// 1. Sets up by creating 99 child POs
			// 2. Attempts to create the 100th child PO
			// 3. Cleans up all created child POs
			name: "maximum child POs reached",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "2plsetqdxht7esg")
				r.Set("uid", "f2j5a8vk006baub")
				r.Set("type", "One-Time")
				r.Set("date", "2024-01-01")
				r.Set("division", "ngpjzurmkrfl8fo")
				r.Set("description", "Test description")
				r.Set("total", 100.0)
				r.Set("payment_type", "OnAccount")
				r.Set("vendor", "2zqxtsmymf670ha")
				r.Set("approver", "wegviunlyr2jjjv")
				r.Set("status", "Unapproved")
				return r
			}(),
			setup: func(t *testing.T, app *tests.TestApp) {
				// Create 99 child POs
				for i := 1; i <= 99; i++ {
					child := core.NewRecord(poCollection)
					child.Set("parent_po", "2plsetqdxht7esg")
					child.Set("po_number", fmt.Sprintf("2024-0008-%02d", i))
					child.Set("uid", "f2j5a8vk006baub")
					child.Set("type", "One-Time")
					child.Set("date", "2024-01-01")
					child.Set("division", "ngpjzurmkrfl8fo")
					child.Set("description", "Test description")
					child.Set("total", 100.0)
					child.Set("approval_total", 100.0)
					child.Set("payment_type", "OnAccount")
					child.Set("vendor", "2zqxtsmymf670ha")
					child.Set("approver", "wegviunlyr2jjjv")
					child.Set("status", "Unapproved")
					child.Set("kind", utilities.DefaultExpenditureKindID())
					if err := app.Save(child); err != nil {
						t.Fatalf("failed to save child PO %d: %v", i, err)
					}
				}
			},
			cleanup: func(t *testing.T, app *tests.TestApp) {
				// Delete all child POs
				records, err := app.FindRecordsByFilter(
					"purchase_orders",
					"parent_po = {:parentId}",
					"",
					0,
					0,
					dbx.Params{"parentId": "2plsetqdxht7esg"},
				)
				if err != nil {
					t.Fatalf("failed to find child POs to clean up: %v", err)
				}
				for _, record := range records {
					if err := app.Delete(record); err != nil {
						t.Fatalf("failed to delete child PO: %v", err)
					}
				}
			},
			expectedError: "maximum number of child POs reached (99) for parent 2024-0008",
		},
		{
			// Test Case 6: First PO of the year
			// This test verifies that when creating the first PO of the year:
			// - No existing POs are found
			// - The generated number follows the format: YYMM + "-0001"
			// - The number is unique in the database
			name: "first PO of the year",
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			setup: func(t *testing.T, app *tests.TestApp) {
				// Delete all POs from current period to ensure we're starting fresh
				records, err := app.FindRecordsByFilter(
					"purchase_orders",
					`po_number ~ '{:prefix}-%'`,
					"",
					0,
					0,
					dbx.Params{"prefix": currentPoPrefix},
				)
				if err != nil {
					t.Fatalf("failed to find current period POs: %v", err)
				}
				for _, record := range records {
					if err := app.Delete(record); err != nil {
						t.Fatalf("failed to delete existing PO: %v", err)
					}
				}
			},
			expected: fmt.Sprintf("%s0001", currentPoPrefix),
		},
		{
			// Test Case 7: Parent PO not found
			// This test verifies that when creating a child PO:
			// - If the parent PO does not exist
			// - An appropriate error is returned
			name: "parent PO not found",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "nonexistent")
				return r
			}(),
			expectedError: "parent PO not found",
		},
		{
			// Test Case 8: Manual/import range is ignored for parent sequencing
			// Fixture rows include only 2402-5000 and 2402-5001.
			// Auto-generation should still start at 2402-0001.
			name:  "manual range does not affect first autogenerated parent PO",
			year:  2024,
			month: 2,
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			expected: "2402-0001",
		},
		{
			// Test Case 9: Child rows with the same prefix do not block parent generation.
			// Fixture rows include 2403-0010 (parent) and 2403-0010-01 (child).
			name:  "child with same prefix does not break parent sequence",
			year:  2024,
			month: 3,
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			expected: "2403-0011",
		},
		{
			// Test Case 10: Defensive parsing handles dashed suffixes.
			// Fixture row includes malformed parent-style number 2404-0010-01.
			name:  "dashed suffix in max record is parsed defensively",
			year:  2024,
			month: 4,
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			expected: "2404-0011",
		},
		{
			// Test Case 11: Upper boundary still allows 4999.
			name:  "generates 4999 when 4998 exists",
			year:  2024,
			month: 5,
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			expected: "2405-4999",
		},
		{
			// Test Case 12: Range exhaustion at 4999.
			name:  "returns error when 4999 already exists",
			year:  2024,
			month: 6,
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			expectedError: "unable to generate a unique PO number",
		},
		{
			// Test Case 13: Manual 5000+ entries do not shadow sub-5000 max.
			// Fixture rows include 2407-4998 and 2407-5000.
			name:  "manual 5000 plus is ignored when deriving max below threshold",
			year:  2024,
			month: 7,
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			expected: "2407-4999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t, app)
			}

			if tt.cleanup != nil {
				defer tt.cleanup(t, app)
			}

			var result string
			var err error
			if tt.year != 0 {
				result, err = routes.GeneratePONumber(app, tt.record, tt.year, tt.month)
			} else {
				// default to current year/month
				result, err = routes.GeneratePONumber(app, tt.record)
			}

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectedError)
					return
				}
				if err.Error() != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
