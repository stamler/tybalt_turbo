/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "amfas3ce",
    "name": "hours",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 0,
      "max": 18,
      "noDecimal": false
    }
  }))

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "amfas3ce",
    "name": "hours",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 0,
      "max": 180,
      "noDecimal": true
    }
  }))

  return dao.saveCollection(collection)
})
