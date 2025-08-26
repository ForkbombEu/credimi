/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // update field
  collection.fields.addAt(11, new Field({
    "hidden": false,
    "id": "json28711958692",
    "maxSize": 0,
    "name": "input_json_schema",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // update field
  collection.fields.addAt(11, new Field({
    "hidden": false,
    "id": "json28711958692",
    "maxSize": 0,
    "name": "input_json_schema",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "json"
  }))

  return app.save(collection)
})
