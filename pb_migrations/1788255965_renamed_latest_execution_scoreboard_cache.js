/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pipeline_scoreboard_cache")

  // rename latest_successful_execution -> latest_execution
  // (same field id, so stored relation values are preserved)
  collection.fields.addAt(12, new Field({
    "cascadeDelete": true,
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
}, (app) => {
  const collection = app.findCollectionByNameOrId("pipeline_scoreboard_cache")

  // restore latest_successful_execution
  collection.fields.addAt(12, new Field({
    "cascadeDelete": true,
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
})
