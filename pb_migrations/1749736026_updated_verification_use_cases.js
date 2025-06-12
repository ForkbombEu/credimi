/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_92944219")

  // add field
  collection.fields.addAt(7, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_183765882",
    "hidden": false,
    "id": "relation4194641934",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "credentials",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_92944219")

  // remove field
  collection.fields.removeById("relation4194641934")

  return app.save(collection)
})
