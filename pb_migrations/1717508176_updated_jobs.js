/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("yovqzrnnomp0lkx")

  collection.name = "Jobs"

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("yovqzrnnomp0lkx")

  collection.name = "jobs"

  return dao.saveCollection(collection)
})
