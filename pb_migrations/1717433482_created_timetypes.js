/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "cnqv0wm8hly7r3n",
    "created": "2024-06-03 16:51:22.959Z",
    "updated": "2024-06-03 16:51:22.959Z",
    "name": "timetypes",
    "type": "base",
    "system": false,
    "schema": [
      {
        "system": false,
        "id": "eoitnxlx",
        "name": "code",
        "type": "text",
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": 2,
          "max": null,
          "pattern": ""
        }
      },
      {
        "system": false,
        "id": "rwphtkdf",
        "name": "name",
        "type": "text",
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": 2,
          "max": null,
          "pattern": ""
        }
      }
    ],
    "indexes": [
      "CREATE UNIQUE INDEX `idx_fQtszvd` ON `timetypes` (`code`)"
    ],
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
  const collection = dao.findCollectionByNameOrId("cnqv0wm8hly7r3n");

  return dao.deleteCollection(collection);
})
