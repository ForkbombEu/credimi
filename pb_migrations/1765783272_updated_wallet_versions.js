/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2201295156")

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "bool2053148143",
    "name": "downloadable",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2201295156")

  // remove field
  collection.fields.removeById("bool2053148143")

  return app.save(collection)
})
