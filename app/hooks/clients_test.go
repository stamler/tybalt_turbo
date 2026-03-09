package hooks

import (
	"testing"
	"tybalt/internal/testseed"

	"github.com/pocketbase/pocketbase/core"
)

// Two integration-style tests exercising ProcessClient with the real test DB.

func TestProcessClient_BusinessDevelopmentLeadClaimSuccess(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	// user with busdev claim from test DB
	leadID := "4r70mfovf22m9uh"

	clientsColl, _ := app.FindCollectionByNameOrId("clients")
	rec := core.NewRecord(clientsColl)
	rec.Set("business_development_lead", leadID)

	evt := &core.RecordRequestEvent{Record: rec}
	if err := ProcessClient(app, evt); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestProcessClient_BusinessDevelopmentLeadClaimFailure(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	// user without busdev claim from test DB
	leadID := "4ssj9f1yg250o9y"

	clientsColl, _ := app.FindCollectionByNameOrId("clients")
	rec := core.NewRecord(clientsColl)
	rec.Set("business_development_lead", leadID)

	evt := &core.RecordRequestEvent{Record: rec}
	if err := ProcessClient(app, evt); err == nil {
		t.Fatalf("expected failure, got success")
	}
}
