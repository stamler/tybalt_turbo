package routes

import (
	"testing"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
)

// A value change's review token describes the related records a response
// shows, even when one is edited before the token is calculated.
func TestValueReviewTokenDescribesShownRelatedData(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	job, err := app.FindRecordById("jobs", "pawflegacy00001")
	if err != nil {
		t.Fatal(err)
	}
	shownToken, err := valueReviewToken(app, job)
	if err != nil {
		t.Fatal(err)
	}
	shown, err := requestRelated(app, job)
	if err != nil {
		t.Fatal(err)
	}
	// An invoicing profile edit lands after the response read its related records.
	if _, err = app.DB().NewQuery("UPDATE client_invoicing_information SET invoicing_instructions = 'Changed after reading' WHERE id = {:id}").Bind(dbx.Params{"id": job.GetString("invoicing_information")}).Execute(); err != nil {
		t.Fatal(err)
	}
	token, err := valueReviewTokenFor(app, job, shown)
	if err != nil {
		t.Fatal(err)
	}
	if token != shownToken {
		t.Fatal("the review token describes data the response did not show")
	}
	if current, err := valueReviewToken(app, job); err != nil || current == shownToken {
		t.Fatalf("the profile edit did not change the current token: %v", err)
	}
}
