/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a")

  collection.name = "time_entries"

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ranctx5xgih6n3a")

  collection.name = "TimeEntries"

  return dao.saveCollection(collection)
})
