/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "svbnxyon",
    "name": "date",
    "type": "date",
    "required": true,
    "presentable": false,
    "unique": false,
    "options": {
      "min": "2024-06-01 08:00:00.000Z",
      "max": "2050-05-31 08:00:00.000Z"
    }
  }))

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
      "max": 3,
      "noDecimal": false
    }
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "xio2lxq5",
    "name": "job_hours",
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

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "n8ys3o83",
    "name": "payout_request_amount",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": null,
      "max": null,
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
    "id": "svbnxyon",
    "name": "date",
    "type": "date",
    "required": true,
    "presentable": false,
    "unique": false,
    "options": {
      "min": "2024-06-01 12:00:00.000Z",
      "max": "2050-05-31 12:00:00.000Z"
    }
  }))

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

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "xio2lxq5",
    "name": "job_hours",
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

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "n8ys3o83",
    "name": "payout_request_amount",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": null,
      "max": null,
      "noDecimal": true
    }
  }))

  return dao.saveCollection(collection)
})
