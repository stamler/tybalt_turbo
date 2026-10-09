package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Give clients an optional alias, the name they are commonly known by, beside
// their official (often numbered) company name. Each invoicing profile chooses
// which of the two names its documents bill. A blank billing_name means the
// official name, so existing profiles and code that creates profiles need no
// backfill. Only fields are added: collection rules are left unchanged.
func init() {
	m.Register(func(app core.App) error {
		clients, err := app.FindCollectionByNameOrId("clients")
		if err != nil {
			return err
		}
		clients.Fields.Add(&core.TextField{
			Id:   "text1791559630",
			Name: "alias",
			Max:  120,
		})
		if err := app.Save(clients); err != nil {
			return err
		}

		profiles, err := app.FindCollectionByNameOrId("client_invoicing_information")
		if err != nil {
			return err
		}
		profiles.Fields.Add(&core.SelectField{
			Id:        "select1791559630",
			Name:      "billing_name",
			MaxSelect: 1,
			Values:    []string{"name", "alias"},
		})
		return app.Save(profiles)
	}, func(app core.App) error {
		profiles, err := app.FindCollectionByNameOrId("client_invoicing_information")
		if err != nil {
			return err
		}
		profiles.Fields.RemoveByName("billing_name")
		if err := app.Save(profiles); err != nil {
			return err
		}

		clients, err := app.FindCollectionByNameOrId("clients")
		if err != nil {
			return err
		}
		clients.Fields.RemoveByName("alias")
		return app.Save(clients)
	})
}
