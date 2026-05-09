/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // update field
  collection.fields.addAt(9, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2980015441",
    "hidden": false,
    "id": "relation3634082342",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "latest_successful_execution",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // update field
  collection.fields.addAt(9, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2980015441",
    "hidden": false,
    "id": "relation3634082342",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "latest_execution",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
})
