/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "yovqzrnnomp0lkx",
    "created": "2024-03-24 14:50:35.856Z",
    "updated": "2024-03-24 14:50:35.856Z",
    "name": "jobs",
    "type": "base",
    "system": false,
    "schema": [
      {
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
  const collection = dao.findCollectionByNameOrId("yovqzrnnomp0lkx");

  return dao.deleteCollection(collection);
})
