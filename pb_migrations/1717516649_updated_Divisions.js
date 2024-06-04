/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("3esdddggow6dykr")

  collection.name = "divisions"
  collection.indexes = [
    "CREATE UNIQUE INDEX `idx_rbNPJNF` ON `divisions` (`code`)"
  ]

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("3esdddggow6dykr")

  collection.name = "Divisions"
  collection.indexes = [
    "CREATE UNIQUE INDEX `idx_rbNPJNF` ON `Divisions` (`code`)"
  ]

  return dao.saveCollection(collection)
})
