/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("yovqzrnnomp0lkx")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "zloyds7s",
    "name": "job_number",
    "type": "text",
    "required": true,
    "presentable": true,
    "unique": false,
    "options": {
      "min": null,
      "max": null,
      "pattern": "(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?"
    }
  }))

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("yovqzrnnomp0lkx")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "zloyds7s",
    "name": "job_number",
    "type": "text",
    "required": true,
    "presentable": true,
    "unique": false,
    "options": {
      "min": null,
      "max": null,
      "pattern": ""
    }
  }))

  return dao.saveCollection(collection)
})
