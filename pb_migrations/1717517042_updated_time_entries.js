/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "r18fowxw",
    "name": "meals_hours",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 0,
      "max": 30,
      "noDecimal": true
    }
  }))

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "r18fowxw",
    "name": "mealsHours",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 0,
      "max": 30,
      "noDecimal": true
    }
  }))

  return dao.saveCollection(collection)
})
