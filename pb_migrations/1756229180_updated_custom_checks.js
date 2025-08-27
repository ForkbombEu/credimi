/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // update field
  collection.fields.addAt(10, new Field({
    "hidden": false,
    "id": "json2871195869",
    "maxSize": 0,
    "name": "input_json_sample",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // update field
  collection.fields.addAt(10, new Field({
    "hidden": false,
    "id": "json2871195869",
    "maxSize": 0,
    "name": "input_json_sample",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "json"
  }))

  return app.save(collection)
})
