package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("c9b90wqyjpqa7tk")
		if err != nil {
			return err
		}

		collection.Name = "payroll_year_end_dates"

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("c9b90wqyjpqa7tk")
		if err != nil {
			return err
		}

		collection.Name = "time_off_reset_dates"

		return dao.SaveCollection(collection)
	})
}
