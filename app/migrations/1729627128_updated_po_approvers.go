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

		collection, err := dao.FindCollectionByNameOrId("kn6f5sfmzjogw63")
		if err != nil {
			return err
		}

		options := map[string]any{}
		if err := json.Unmarshal([]byte(`{
			"query": "SELECT p.uid AS id, p.surname AS surname, p.given_name AS given_name, u.payload as divisions \nFROM profiles p\nINNER JOIN user_claims u ON p.uid = u.uid\nINNER JOIN claims c ON u.cid = c.id\nWHERE c.name = 'po_approver'"
		}`), &options); err != nil {
			return err
		}
		collection.SetOptions(options)

		// remove
		collection.Schema.RemoveField("pqu2ncyf")

		// remove
		collection.Schema.RemoveField("0onwdwi1")

		// add
		new_surname := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "vtlc0zc3",
			"name": "surname",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 2,
				"max": 48,
				"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$"
			}
		}`), new_surname); err != nil {
			return err
		}
		collection.Schema.AddField(new_surname)

		// add
		new_given_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "lnnexcis",
			"name": "given_name",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 2,
				"max": 48,
				"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$"
			}
		}`), new_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(new_given_name)

		// add
		new_divisions := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "l76zxzcc",
			"name": "divisions",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 2000000
			}
		}`), new_divisions); err != nil {
			return err
		}
		collection.Schema.AddField(new_divisions)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("kn6f5sfmzjogw63")
		if err != nil {
			return err
		}

		options := map[string]any{}
		if err := json.Unmarshal([]byte(`{
			"query": "SELECT p.uid AS id, p.surname AS surname, p.given_name AS given_name \nFROM profiles p\nINNER JOIN user_claims u ON p.uid = u.uid\nINNER JOIN claims c ON u.cid = c.id\nWHERE c.name = 'po_approver'"
		}`), &options); err != nil {
			return err
		}
		collection.SetOptions(options)

		// add
		del_surname := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "pqu2ncyf",
			"name": "surname",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 2,
				"max": 48,
				"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$"
			}
		}`), del_surname); err != nil {
			return err
		}
		collection.Schema.AddField(del_surname)

		// add
		del_given_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "0onwdwi1",
			"name": "given_name",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 2,
				"max": 48,
				"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$"
			}
		}`), del_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(del_given_name)

		// remove
		collection.Schema.RemoveField("vtlc0zc3")

		// remove
		collection.Schema.RemoveField("lnnexcis")

		// remove
		collection.Schema.RemoveField("l76zxzcc")

		return dao.SaveCollection(collection)
	})
}
