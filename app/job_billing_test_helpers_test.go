package main

import (
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

// Existing job tests focus on status, dates, or other fields. Complete only the
// new billing relation from CSV fixtures so those tests reach their own checks.
// This changes no value, status, date, client, contact, or approval evidence.
func setupJobsWithBillingFixtures(t testing.TB) *tests.TestApp {
	t.Helper()
	app := testutils.SetupTestApp(t)
	if _, err := app.DB().NewQuery(`UPDATE jobs SET invoicing_information =
 (SELECT id FROM client_invoicing_information WHERE client = jobs.client ORDER BY id LIMIT 1)
 WHERE invoicing_information = '' AND EXISTS
 (SELECT 1 FROM client_invoicing_information WHERE client = jobs.client)`).Execute(); err != nil {
		t.Fatal(err)
	}
	return app
}
