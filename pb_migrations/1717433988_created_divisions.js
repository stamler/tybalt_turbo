/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "3esdddggow6dykr",
    "created": "2024-06-03 16:59:48.189Z",
    "updated": "2024-06-03 16:59:48.189Z",
    "name": "divisions",
    "type": "base",
    "system": false,
    "schema": [
      {
        "system": false,
        "id": "cmlhnbq8",
        "name": "code",
        "type": "text",
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": 1,
          "max": null,
          "pattern": ""
        }
      },
      {
        "system": false,
        "id": "xc9wslmg",
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
      "CREATE UNIQUE INDEX `idx_rbNPJNF` ON `divisions` (`code`)"
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
  const collection = dao.findCollectionByNameOrId("3esdddggow6dykr");

  return dao.deleteCollection(collection);
})
