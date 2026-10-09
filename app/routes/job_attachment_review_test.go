package routes

import (
	"testing"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
)

// A value change's review token describes the data a response read and shows,
// even when a related record is edited before the token is calculated.
func TestValueReviewTokenDescribesShownData(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	job, err := app.FindRecordById("jobs", "pawflegacy00001")
	if err != nil {
		t.Fatal(err)
	}
	shown, err := readValueReview(app, job)
	if err != nil {
		t.Fatal(err)
	}
	before, err := valueReviewToken(app, job)
	if err != nil {
		t.Fatal(err)
	}
	// An invoicing profile edit lands after the response read its review data.
	if _, err = app.DB().NewQuery("UPDATE client_invoicing_information SET invoicing_instructions = 'Changed after reading' WHERE id = {:id}").Bind(dbx.Params{"id": job.GetString("invoicing_information")}).Execute(); err != nil {
		t.Fatal(err)
	}
	if token, err := shown.token(); err != nil || token != before {
		t.Fatalf("the review token describes data the response did not show: %v", err)
	}
	if current, err := valueReviewToken(app, job); err != nil || current == before {
		t.Fatalf("the profile edit did not change the current token: %v", err)
	}
}
