/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_500646217")

  // add field
  collection.fields.addAt(14, new Field({
    "hidden": false,
    "id": "bool2231267043",
    "name": "disabled",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_500646217")

  // remove field
  collection.fields.removeById("bool2231267043")

  return app.save(collection)
})
