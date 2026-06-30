/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  // add field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "bool1748787223",
    "name": "published",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  // remove field
  collection.fields.removeById("bool1748787223")

  return app.save(collection)
})
