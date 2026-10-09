package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Give project requests optional notes, such as a survey property's legal
// description. Staff can print them on the Project Authorization, and job
// creation copies them into the job's first client note, so they use the same
// 10 to 1000 character limits as client_notes.note. Only a field is added:
// collection rules are left unchanged.
func init() {
	m.Register(func(app core.App) error {
		requests, err := app.FindCollectionByNameOrId("job_requests")
		if err != nil {
			return err
		}
		requests.Fields.Add(&core.TextField{
			Id:   "text1791570899",
			Name: "notes",
			Min:  10,
			Max:  1000,
		})
		return app.Save(requests)
	}, func(app core.App) error {
		requests, err := app.FindCollectionByNameOrId("job_requests")
		if err != nil {
			return err
		}
		requests.Fields.RemoveByName("notes")
		return app.Save(requests)
	})
}
