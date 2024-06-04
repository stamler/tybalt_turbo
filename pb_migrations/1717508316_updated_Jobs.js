/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("yovqzrnnomp0lkx")

  collection.indexes = [
    "CREATE UNIQUE INDEX `idx_V1RKd7H` ON `Jobs` (`job_number`)"
  ]

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("yovqzrnnomp0lkx")

  collection.indexes = []

  return dao.saveCollection(collection)
})
