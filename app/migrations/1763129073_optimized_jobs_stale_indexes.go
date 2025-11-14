package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds indexes to optimize the jobs_stale query performance
func init() {
	m.Register(func(app core.App) error {
		// Add index on time_amendments(job)
		collection, err := app.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		// set the complete indexes list (existing + new)
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX `+"`"+`idx_Yq0DnuvO5A`+"`"+` ON `+"`"+`time_amendments`+"`"+` (\n  `+"`"+`uid`+"`"+`,\n  `+"`"+`committed_week_ending`+"`"+`\n) WHERE committed != ''",
				"CREATE INDEX `+"`"+`idx_time_amendments_job`+"`"+` ON `+"`"+`time_amendments`+"`"+` (`+"`"+`job`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_time_amendments_job_date`+"`"+` ON `+"`"+`time_amendments`+"`"+` (\n  `+"`"+`job`+"`"+`,\n  `+"`"+`date`+"`"+`\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		// Add index on jobs(status)
		collection, err = app.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// set the complete indexes list (existing + new)
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_V1RKd7H`+"`"+` ON `+"`"+`jobs`+"`"+` (`+"`"+`number`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_d1R7JSCuuJ`+"`"+` ON `+"`"+`jobs`+"`"+` (`+"`"+`proposal`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_SCUCvwyln3`+"`"+` ON `+"`"+`jobs`+"`"+` (`+"`"+`parent`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_jobs_status`+"`"+` ON `+"`"+`jobs`+"`"+` (`+"`"+`status`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Remove the added indexes in reverse order

		// Remove jobs status index
		collection, err := app.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_V1RKd7H`+"`"+` ON `+"`"+`jobs`+"`"+` (`+"`"+`number`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_d1R7JSCuuJ`+"`"+` ON `+"`"+`jobs`+"`"+` (`+"`"+`proposal`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_SCUCvwyln3`+"`"+` ON `+"`"+`jobs`+"`"+` (`+"`"+`parent`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		// Remove time_amendments indexes
		collection, err = app.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX `+"`"+`idx_Yq0DnuvO5A`+"`"+` ON `+"`"+`time_amendments`+"`"+` (\n  `+"`"+`uid`+"`"+`,\n  `+"`"+`committed_week_ending`+"`"+`\n) WHERE committed != ''"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
