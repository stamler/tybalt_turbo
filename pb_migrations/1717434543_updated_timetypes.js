/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("cnqv0wm8hly7r3n")

  collection.listRule = "  @request.auth.id != \"\""

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("cnqv0wm8hly7r3n")

  collection.listRule = null

  return dao.saveCollection(collection)
})
