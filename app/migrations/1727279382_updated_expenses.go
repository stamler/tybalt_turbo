package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// add
		new_pay_period_ending := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "6ocqzyet",
			"name": "pay_period_ending",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
			}
		}`), new_pay_period_ending); err != nil {
			return err
		}
		collection.Schema.AddField(new_pay_period_ending)

		// add
		new_allowance_types := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tahxw786",
			"name": "allowance_types",
			"type": "select",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 2,
				"values": [
					"Lodging",
					"Breakfast",
					"Lunch",
					"Dinner"
				]
			}
		}`), new_allowance_types); err != nil {
			return err
		}
		collection.Schema.AddField(new_allowance_types)

		// add
		new_submitted := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "cpt1x5gr",
			"name": "submitted",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_submitted); err != nil {
			return err
		}
		collection.Schema.AddField(new_submitted)

		// add
		new_committer := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "djy3zkz8",
			"name": "committer",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "_pb_users_auth_",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_committer); err != nil {
			return err
		}
		collection.Schema.AddField(new_committer)

		// add
		new_committed := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "bmzx8tgn",
			"name": "committed",
			"type": "date",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": "",
				"max": ""
			}
		}`), new_committed); err != nil {
			return err
		}
		collection.Schema.AddField(new_committed)

		// add
		new_committed_week_ending := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "d13a8jxo",
			"name": "committed_week_ending",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
			}
		}`), new_committed_week_ending); err != nil {
			return err
		}
		collection.Schema.AddField(new_committed_week_ending)

		// add
		new_distance := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "hsvbnev9",
			"name": "distance",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 0,
				"max": null,
				"noDecimal": false
			}
		}`), new_distance); err != nil {
			return err
		}
		collection.Schema.AddField(new_distance)

		// add
		new_cc_last_4_digits := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "gv2z62zj",
			"name": "cc_last_4_digits",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^\\d{4}$"
			}
		}`), new_cc_last_4_digits); err != nil {
			return err
		}
		collection.Schema.AddField(new_cc_last_4_digits)

		// add
		new_purchase_order := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "pxd0mvyh",
			"name": "purchase_order",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "m19q72syy0e3lvm",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_purchase_order); err != nil {
			return err
		}
		collection.Schema.AddField(new_purchase_order)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("6ocqzyet")

		// remove
		collection.Schema.RemoveField("tahxw786")

		// remove
		collection.Schema.RemoveField("cpt1x5gr")

		// remove
		collection.Schema.RemoveField("djy3zkz8")

		// remove
		collection.Schema.RemoveField("bmzx8tgn")

		// remove
		collection.Schema.RemoveField("d13a8jxo")

		// remove
		collection.Schema.RemoveField("hsvbnev9")

		// remove
		collection.Schema.RemoveField("gv2z62zj")

		// remove
		collection.Schema.RemoveField("pxd0mvyh")

		return dao.SaveCollection(collection)
	})
}
