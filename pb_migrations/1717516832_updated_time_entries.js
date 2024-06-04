/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "xkbfo3ev",
    "name": "time_type",
    "type": "relation",
    "required": true,
    "presentable": false,
    "unique": false,
    "options": {
      "collectionId": "cnqv0wm8hly7r3n",
      "cascadeDelete": false,
      "minSelect": null,
      "maxSelect": 1,
      "displayFields": null
    }
  }))

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "xkbfo3ev",
    "name": "timetype",
    "type": "relation",
    "required": true,
    "presentable": false,
    "unique": false,
    "options": {
      "collectionId": "cnqv0wm8hly7r3n",
      "cascadeDelete": false,
      "minSelect": null,
      "maxSelect": 1,
      "displayFields": null
    }
  }))

  return dao.saveCollection(collection)
})
