/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("cnqv0wm8hly7r3n")

  // add
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "q4ppqv3i",
    "name": "description",
    "type": "text",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": null,
      "max": null,
      "pattern": ""
    }
  }))

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("cnqv0wm8hly7r3n")

  // remove
  collection.schema.removeField("q4ppqv3i")

  return dao.saveCollection(collection)
})
