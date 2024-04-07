/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("yovqzrnnomp0lkx")

  collection.createRule = "@request.auth.id = \"f2j5a8vk006baub\""

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("yovqzrnnomp0lkx")

  collection.createRule = "@request.auth.id != \"f2j5a8vk006baub\""

  return dao.saveCollection(collection)
})
