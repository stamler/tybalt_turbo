/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "ranctx5xgih6n3a",
    "created": "2024-06-04 13:35:40.992Z",
    "updated": "2024-06-04 13:35:40.992Z",
    "name": "TimeEntries",
    "type": "base",
    "system": false,
    "schema": [
      {
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
      },
      {
        "system": false,
        "id": "jlqkb6jb",
        "name": "division",
        "type": "relation",
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "collectionId": "3esdddggow6dykr",
          "cascadeDelete": false,
          "minSelect": null,
          "maxSelect": 1,
          "displayFields": null
        }
      },
      {
        "system": false,
        "id": "rjasv0rb",
        "name": "uid",
        "type": "relation",
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "collectionId": "_pb_users_auth_",
          "cascadeDelete": false,
          "minSelect": null,
          "maxSelect": 1,
          "displayFields": null
        }
      },
      {
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
      },
      {
        "system": false,
        "id": "4eu16q2p",
        "name": "description",
        "type": "text",
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": 5,
          "max": null,
          "pattern": ""
        }
      },
      {
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
      },
      {
        "system": false,
        "id": "lriva8hh",
        "name": "weekEnding",
        "type": "date",
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": "",
          "max": ""
        }
      },
      {
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
      },
      {
        "system": false,
        "id": "xio2lxq5",
        "name": "jobHours",
        "type": "number",
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": 0,
          "max": 180,
          "noDecimal": true
        }
      },
      {
        "system": false,
        "id": "jcncwdjc",
        "name": "job",
        "type": "relation",
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "collectionId": "yovqzrnnomp0lkx",
          "cascadeDelete": false,
          "minSelect": null,
          "maxSelect": 1,
          "displayFields": null
        }
      },
      {
        "system": false,
        "id": "fjcrzqdc",
        "name": "workrecord",
        "type": "text",
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$"
        }
      },
      {
        "system": false,
        "id": "ymg43f6u",
        "name": "category",
        "type": "text",
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
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
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a");

  return dao.deleteCollection(collection);
})
