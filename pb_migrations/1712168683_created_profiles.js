/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "glmf9xpnwgpwudm",
    "created": "2024-04-03 18:24:43.543Z",
    "updated": "2024-04-03 18:24:43.543Z",
    "name": "profiles",
    "type": "base",
    "system": false,
    "schema": [
      {
        "system": false,
        "id": "fxlkxvsy",
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
      },
      {
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
      },
      {
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
      },
      {
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
      },
      {
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
      },
      {
        "system": false,
        "id": "v6thasef",
        "name": "untrackedTimeOff",
        "type": "bool",
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {}
      },
      {
        "system": false,
        "id": "9qfzn9ab",
        "name": "timestamp",
        "type": "date",
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": "",
          "max": ""
        }
      },
      {
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
      }
    ],
    "indexes": [],
    "listRule": null,
    "viewRule": null,
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {}
  });

  return Dao(db).saveCollection(collection);
}, (db) => {
  const dao = new Dao(db);
  const collection = dao.findCollectionByNameOrId("glmf9xpnwgpwudm");

  return dao.deleteCollection(collection);
})
