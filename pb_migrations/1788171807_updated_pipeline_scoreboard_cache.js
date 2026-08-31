/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // add field
  collection.fields.addAt(21, new Field({
    "hidden": false,
    "id": "json1123437412",
    "maxSize": 0,
    "name": "expanded_data",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // remove field
  collection.fields.removeById("json1123437412")

  return app.save(collection)
})
