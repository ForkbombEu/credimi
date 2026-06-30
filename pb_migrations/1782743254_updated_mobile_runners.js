/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_500646217")

  // add field
  collection.fields.addAt(10, new Field({
    "hidden": false,
    "id": "bool3402371681",
    "name": "admin_managed",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_500646217")

  // remove field
  collection.fields.removeById("bool3402371681")

  return app.save(collection)
})
