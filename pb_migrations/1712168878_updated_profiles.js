/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("glmf9xpnwgpwudm")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "e7uz2a2n",
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
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "ocxmutn0",
    "name": "opening_datetime_off",
    "type": "text",
    "required": true,
    "presentable": false,
    "unique": false,
    "options": {
      "min": null,
      "max": null,
      "pattern": "^(?:\\d{4})-(?:0[1-9]|1[0-2])-(?:0[1-9]|[1-2][0-9]|3[0-1])$"
    }
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "mghymcxc",
    "name": "opening_op",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 0,
      "max": 10000,
      "noDecimal": true
    }
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "suw6v59k",
    "name": "opening_ov",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 0,
      "max": 100000,
      "noDecimal": true
    }
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "v6thasef",
    "name": "untracked_time_off",
    "type": "bool",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {}
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "65fcj2kd",
    "name": "default_charge_out_rate",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 5000,
      "max": 100000,
      "noDecimal": true
    }
  }))

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("glmf9xpnwgpwudm")

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "e7uz2a2n",
    "name": "givenName",
    "type": "text",
    "required": true,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 2,
      "max": 48,
      "pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$"
    }
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "ocxmutn0",
    "name": "openingDateTimeOff",
    "type": "text",
    "required": true,
    "presentable": false,
    "unique": false,
    "options": {
      "min": null,
      "max": null,
      "pattern": "^(?:\\d{4})-(?:0[1-9]|1[0-2])-(?:0[1-9]|[1-2][0-9]|3[0-1])$"
    }
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "mghymcxc",
    "name": "openingOP",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 0,
      "max": 10000,
      "noDecimal": true
    }
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "suw6v59k",
    "name": "openingOV",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 0,
      "max": 100000,
      "noDecimal": true
    }
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "v6thasef",
    "name": "untrackedTimeOff",
    "type": "bool",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {}
  }))

  // update
  collection.schema.addField(new SchemaField({
    "system": false,
    "id": "65fcj2kd",
    "name": "defaultChargeOutRate",
    "type": "number",
    "required": false,
    "presentable": false,
    "unique": false,
    "options": {
      "min": 5000,
      "max": 100000,
      "noDecimal": true
    }
  }))

  return dao.saveCollection(collection)
})
