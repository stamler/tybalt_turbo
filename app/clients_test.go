package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestClientsCreate_BusDevLeadMissingClaim_Fails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "client create fails when business_development_lead lacks busdev claim",
			Method: http.MethodPost,
			URL:    "/api/collections/clients/records",
			Body: strings.NewReader(`{
                "name": "Acme Widgets",
                "business_development_lead": "4ssj9f1yg250o9y"
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"business_development_lead":{"code":"missing_claim"`,
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

func TestClientsCreate_BusDevLeadWithClaim_Succeeds(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "client create succeeds when business_development_lead has busdev claim",
			Method: http.MethodPost,
			URL:    "/api/collections/clients/records",
			Body: strings.NewReader(`{
                "name": "Globex Corporation",
                "business_development_lead": "4r70mfovf22m9uh"
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"name":"Globex Corporation"`,
				`"business_development_lead":"4r70mfovf22m9uh"`,
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

func TestClientsCreate_BusDevLeadInactive_Fails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "client create fails when business_development_lead is inactive",
			Method: http.MethodPost,
			URL:    "/api/collections/clients/records",
			Body: strings.NewReader(`{
                "name": "Inactive Lead Corp",
                "business_development_lead": "u_inactive"
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"business_development_lead":{"code":"business_development_lead_not_active"`,
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

// expectClientsReferenced returns an AfterTestFunc that checks the referenced
// flag of each listed client in a GET /api/clients response.
func expectClientsReferenced(want map[string]bool) func(testing.TB, *tests.TestApp, *http.Response) {
	return func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
		defer res.Body.Close()
		var rows []struct {
			ID         string `json:"id"`
			Referenced bool   `json:"referenced"`
		}
		if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
			tb.Fatalf("failed to decode clients: %v", err)
		}
		got := map[string]bool{}
		for _, row := range rows {
			got[row.ID] = row.Referenced
		}
		for id, referenced := range want {
			if g, ok := got[id]; !ok {
				tb.Fatalf("client %s missing from the list", id)
			} else if g != referenced {
				tb.Fatalf("client %s referenced = %v, want %v", id, g, referenced)
			}
		}
	}
}

// clientUse is one way a record can use the otherwise unused client
// cwclient0000002.
type clientUse struct {
	collection string
	field      string
	// query makes one fixture row use the client.
	query string
	// deleteStatus is how the server refuses to delete the client: 404 when the
	// clients deleteRule refuses it, 400 when the delete hook does.
	deleteStatus int
}

var clientUses = []clientUse{
	{"jobs", "client", "UPDATE jobs SET client = 'cwclient0000002' WHERE id = 'cjf0kt0defhq480'", http.StatusNotFound},
	{"jobs", "job_owner", "UPDATE jobs SET job_owner = 'cwclient0000002' WHERE id = 'cjf0kt0defhq480'", http.StatusBadRequest},
	{"job_requests", "client", "UPDATE job_requests SET client = 'cwclient0000002' WHERE id = 'parequest000001'", http.StatusNotFound},
	{"job_requests", "job_owner", "UPDATE job_requests SET job_owner = 'cwclient0000002' WHERE id = 'parequest000001'", http.StatusBadRequest},
	{"client_invoicing_information", "client", "UPDATE client_invoicing_information SET client = 'cwclient0000002' WHERE id = 'cwinvoice000003'", http.StatusNotFound},
	{"client_notes", "client", "UPDATE client_notes SET client = 'cwclient0000002' WHERE id = 'note_eld'", http.StatusBadRequest},
}

// TestClientsApi_Referenced verifies that the clients API reports whether any
// record uses a client, so the UI can hide Delete, and
// that the server refuses to delete a client for each of those uses.
func TestClientsApi_Referenced(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			// cwclient0000001 is used only by invoicing profiles, and Mobilia SA
			// by jobs, invoicing profiles and notes.
			Name:           "list reports which clients are referenced",
			Method:         http.MethodGet,
			URL:            "/api/clients",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"cwclient0000002"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: expectClientsReferenced(map[string]bool{
				"cwclient0000002": false,
				"cwclient0000001": true,
				"lb0fnenkeyitsny": true,
			}),
		},
		{
			Name:           "unused client is not referenced in its details",
			Method:         http.MethodGet,
			URL:            "/api/clients/cwclient0000002",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"referencing_jobs_count":0`,
				`"referenced":false`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "client used only by invoicing profiles is referenced in its details",
			Method:          http.MethodGet,
			URL:             "/api/clients/cwclient0000001",
			Headers:         map[string]string{"Authorization": recordToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"referenced":true`},
			TestAppFactory:  testutils.SetupTestApp,
		},
	}

	for _, use := range clientUses {
		setup := func(tb testing.TB) *tests.TestApp {
			app := testutils.SetupTestApp(tb)
			if _, err := app.NonconcurrentDB().NewQuery(use.query).Execute(); err != nil {
				tb.Fatal(err)
			}
			return app
		}
		by := use.collection + "." + use.field
		deleteScenario := tests.ApiScenario{
			Name:           "client used only by " + by + " cannot be deleted",
			Method:         http.MethodDelete,
			URL:            "/api/collections/clients/records/cwclient0000002",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: use.deleteStatus,
			TestAppFactory: setup,
		}
		if use.deleteStatus == http.StatusBadRequest {
			deleteScenario.ExpectedContent = []string{`"code":"in_use"`}
		} else {
			deleteScenario.ExpectedContent = []string{`"status":404`}
		}
		scenarios = append(scenarios,
			tests.ApiScenario{
				Name:            "client used only by " + by + " is referenced in the list",
				Method:          http.MethodGet,
				URL:             "/api/clients",
				Headers:         map[string]string{"Authorization": recordToken},
				ExpectedStatus:  200,
				ExpectedContent: []string{`"id":"cwclient0000002"`},
				TestAppFactory:  setup,
				AfterTestFunc:   expectClientsReferenced(map[string]bool{"cwclient0000002": true}),
			},
			tests.ApiScenario{
				Name:            "client used only by " + by + " is referenced in its details",
				Method:          http.MethodGet,
				URL:             "/api/clients/cwclient0000002",
				Headers:         map[string]string{"Authorization": recordToken},
				ExpectedStatus:  200,
				ExpectedContent: []string{`"referenced":true`},
				TestAppFactory:  setup,
			},
			deleteScenario,
		)
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestClientsApi_ReferencedCoversEveryRelation fails when a collection gains a
// relation to clients that clientUses does not exercise, so the referenced flag
// and the delete checks are updated together. Relations that cascade, and
// views, cannot refuse a delete.
func TestClientsApi_ReferencedCoversEveryRelation(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	covered := map[string]bool{}
	for _, use := range clientUses {
		covered[use.collection+"."+use.field] = true
	}

	collections, err := app.FindAllCollections(core.CollectionTypeBase)
	if err != nil {
		t.Fatal(err)
	}
	clients, err := app.FindCollectionByNameOrId("clients")
	if err != nil {
		t.Fatal(err)
	}
	for _, collection := range collections {
		for _, field := range collection.Fields {
			relation, ok := field.(*core.RelationField)
			if !ok || relation.CollectionId != clients.Id || relation.CascadeDelete {
				continue
			}
			if !covered[collection.Name+"."+relation.Name] {
				t.Errorf("%s.%s refers to clients but clientUses does not cover it", collection.Name, relation.Name)
			}
		}
	}
}

// TestClientsApi_JobOwnerCountsAndBalances verifies the job count and balance
// in the clients list when jobs have owners. The fixture jobs have no owners
// and no balances, so this sets both: a job owned by another client counts for
// its owner without adding to the owner's balance, and a job its own client
// owns counts once and adds its balance once.
func TestClientsApi_JobOwnerCountsAndBalances(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:            "job owners count once and do not add to the balance",
		Method:          http.MethodGet,
		URL:             "/api/clients",
		Headers:         map[string]string{"Authorization": recordToken},
		ExpectedStatus:  200,
		ExpectedContent: []string{`"id":"cwclient0000002"`},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app := testutils.SetupTestApp(tb)
			for _, query := range []string{
				"UPDATE jobs SET job_owner = 'cwclient0000002', outstanding_balance = 100 WHERE id = 'cjf0kt0defhq480'",
				"UPDATE jobs SET job_owner = client, outstanding_balance = 50 WHERE id = 'fcprojimpauto01'",
			} {
				if _, err := app.NonconcurrentDB().NewQuery(query).Execute(); err != nil {
					tb.Fatal(err)
				}
			}
			return app
		},
		AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
			defer res.Body.Close()
			var rows []struct {
				ID                   string  `json:"id"`
				ReferencingJobsCount int     `json:"referencing_jobs_count"`
				OutstandingBalance   float64 `json:"outstanding_balance"`
				Referenced           bool    `json:"referenced"`
			}
			if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
				tb.Fatalf("failed to decode clients: %v", err)
			}
			type totals struct {
				count      int
				balance    float64
				referenced bool
			}
			want := map[string]totals{
				// Owns cjf0kt0defhq480, which is billed to Mobilia SA.
				"cwclient0000002": {1, 0, true},
				// Billed for cjf0kt0defhq480 among its 41 jobs.
				"lb0fnenkeyitsny": {41, 100, true},
				// Both billed for and owns fcprojimpauto01.
				"pqpd90fqd5ohjcs": {17, 50, true},
			}
			for _, row := range rows {
				w, ok := want[row.ID]
				if !ok {
					continue
				}
				got := totals{row.ReferencingJobsCount, row.OutstandingBalance, row.Referenced}
				if got != w {
					tb.Errorf("client %s = %+v, want %+v", row.ID, got, w)
				}
				delete(want, row.ID)
			}
			for id := range want {
				tb.Errorf("client %s missing from the list", id)
			}
		},
	}
	scenario.Test(t)
}
