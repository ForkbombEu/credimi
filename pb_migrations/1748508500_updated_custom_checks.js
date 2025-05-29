/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // add field
  collection.fields.addAt(11, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text2871195869",
    "max": 0,
    "min": 0,
    "name": "input_json_schema",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // remove field
  collection.fields.removeById("text2871195869")

  return app.save(collection)
})
