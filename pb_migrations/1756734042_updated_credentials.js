/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_183765882")

  // add field
  collection.fields.addAt(15, new Field({
    "hidden": false,
    "id": "bool2419865273",
    "name": "imported",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_183765882")

  // remove field
  collection.fields.removeById("bool2419865273")

  return app.save(collection)
})
