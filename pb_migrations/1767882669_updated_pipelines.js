/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153234234")

  // update field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "json874646130",
    "maxSize": 0,
    "name": "steps",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153234234")

  // update field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "json874646130",
    "maxSize": 0,
    "name": "steps",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "json"
  }))

  return app.save(collection)
})
