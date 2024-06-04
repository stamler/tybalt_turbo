/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("cnqv0wm8hly7r3n")

  collection.name = "time_types"
  collection.indexes = [
    "CREATE UNIQUE INDEX `idx_fQtszvd` ON `time_types` (`code`)"
  ]

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("cnqv0wm8hly7r3n")

  collection.name = "TimeTypes"
  collection.indexes = [
    "CREATE UNIQUE INDEX `idx_fQtszvd` ON `TimeTypes` (`code`)"
  ]

  return dao.saveCollection(collection)
})
