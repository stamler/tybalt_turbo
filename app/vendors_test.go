package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

// TestVendorsDelete_ReferencedVendorRefused verifies that the vendors delete
// rule refuses to delete a vendor that purchase orders or expenses reference.
// Without the guard PocketBase deletes the vendor and silently blanks the
// vendor field on every referencing record.
func TestVendorsDelete_ReferencedVendorRefused(t *testing.T) {
	// book@keeper.com has the payables_admin claim required for vendor deletion
	recordToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	// assertVendorReferences checks that the vendor still exists and that
	// the collection still has the expected number of records referencing it.
	assertVendorReferences := func(vendorID string, collection string, want int) func(testing.TB, *tests.TestApp, *http.Response) {
		return func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			if _, err := app.FindRecordById("vendors", vendorID); err != nil {
				tb.Fatalf("vendor %s should still exist: %v", vendorID, err)
			}
			records, err := app.FindAllRecords(collection)
			if err != nil {
				tb.Fatal(err)
			}
			got := 0
			for _, record := range records {
				if record.GetString("vendor") == vendorID {
					got++
				}
			}
			if got != want {
				tb.Fatalf("expected %d %s records to reference vendor %s, got %d", want, collection, vendorID, got)
			}
		}
	}

	scenarios := []tests.ApiScenario{
		{
			// z66xe6vqhwtokt4 is referenced by purchase orders but not expenses
			Name:            "vendor referenced by purchase orders cannot be deleted",
			Method:          http.MethodDelete,
			URL:             "/api/collections/vendors/records/z66xe6vqhwtokt4",
			Headers:         map[string]string{"Authorization": recordToken},
			ExpectedStatus:  404,
			ExpectedContent: []string{`"data":{}`},
			ExpectedEvents:  map[string]int{"*": 0},
			TestAppFactory:  testutils.SetupTestApp,
			AfterTestFunc:   assertVendorReferences("z66xe6vqhwtokt4", "purchase_orders", 113),
		},
		{
			// phvendor0000001 is referenced by an expense but not purchase orders
			Name:            "vendor referenced by expenses cannot be deleted",
			Method:          http.MethodDelete,
			URL:             "/api/collections/vendors/records/phvendor0000001",
			Headers:         map[string]string{"Authorization": recordToken},
			ExpectedStatus:  404,
			ExpectedContent: []string{`"data":{}`},
			ExpectedEvents:  map[string]int{"*": 0},
			TestAppFactory:  testutils.SetupTestApp,
			AfterTestFunc:   assertVendorReferences("phvendor0000001", "expenses", 1),
		},
		{
			// ctswqva5onxj75q is not referenced by any purchase order or expense
			Name:           "unreferenced vendor can be deleted",
			Method:         http.MethodDelete,
			URL:            "/api/collections/vendors/records/ctswqva5onxj75q",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 204,
			ExpectedEvents: map[string]int{
				"OnRecordDeleteRequest":      1,
				"OnRecordDelete":             1,
				"OnRecordDeleteExecute":      1,
				"OnRecordAfterDeleteSuccess": 1,
				"OnModelDelete":              1,
				"OnModelDeleteExecute":       1,
				"OnModelAfterDeleteSuccess":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				if _, err := app.FindRecordById("vendors", "ctswqva5onxj75q"); err == nil {
					tb.Fatal("unreferenced vendor should have been deleted")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestVendorsAbsorb_ReferencedVendors verifies that absorbing vendors still
// works with the vendors delete rule. Absorb moves purchase order and expense
// references to the target vendor before it deletes the absorbed vendors.
func TestVendorsAbsorb_ReferencedVendors(t *testing.T) {
	// book@keeper.com has the absorb claim
	recordToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	const targetID = "2zqxtsmymf670ha"
	// z66xe6vqhwtokt4 is referenced by 113 purchase orders; phvendor0000001
	// is referenced by 1 expense. The target starts with 37 purchase orders
	// and 15 expenses.
	countReferences := func(tb testing.TB, app *tests.TestApp, collection string, vendorID string) int {
		records, err := app.FindAllRecords(collection)
		if err != nil {
			tb.Fatal(err)
		}
		count := 0
		for _, record := range records {
			if record.GetString("vendor") == vendorID {
				count++
			}
		}
		return count
	}

	scenario := tests.ApiScenario{
		Name:           "absorbing referenced vendors moves references and deletes them",
		Method:         http.MethodPost,
		URL:            "/api/vendors/" + targetID + "/absorb",
		Body:           strings.NewReader(`{"ids_to_absorb": ["z66xe6vqhwtokt4", "phvendor0000001"]}`),
		Headers:        map[string]string{"Authorization": recordToken},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"message":"Successfully absorbed 2 records into ` + targetID + `"`,
		},
		ExpectedEvents: map[string]int{
			"OnRecordCreate":             1,
			"OnRecordCreateExecute":      1,
			"OnRecordValidate":           1,
			"OnRecordAfterCreateSuccess": 1,
			"OnModelCreate":              1,
			"OnModelCreateExecute":       1,
			"OnModelValidate":            1,
			"OnModelAfterCreateSuccess":  1,
			"*":                          0,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			for _, id := range []string{"z66xe6vqhwtokt4", "phvendor0000001"} {
				if _, err := app.FindRecordById("vendors", id); err == nil {
					tb.Fatalf("absorbed vendor %s should have been deleted", id)
				}
				for _, collection := range []string{"purchase_orders", "expenses"} {
					if got := countReferences(tb, app, collection, id); got != 0 {
						tb.Fatalf("expected no %s to reference absorbed vendor %s, got %d", collection, id, got)
					}
				}
			}
			if got := countReferences(tb, app, "purchase_orders", targetID); got != 37+113 {
				tb.Fatalf("expected %d purchase orders to reference target vendor, got %d", 37+113, got)
			}
			if got := countReferences(tb, app, "expenses", targetID); got != 15+1 {
				tb.Fatalf("expected %d expenses to reference target vendor, got %d", 15+1, got)
			}
		},
	}
	scenario.Test(t)
}
