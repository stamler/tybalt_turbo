package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("jobs")
		if err != nil {
			return err
		}
		// Client details and the client jobs tab select jobs by client.
		jobs.AddIndex("idx_jobs_client", false, "`client`", "")
		return app.Save(jobs)
	}, func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("jobs")
		if err != nil {
			return err
		}
		jobs.RemoveIndex("idx_jobs_client")
		return app.Save(jobs)
	})
}
