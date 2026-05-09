/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // update field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "number4263700706",
    "max": null,
    "min": null,
    "name": "total_runs",
    "onlyInt": false,
    "presentable": false,
    "required": true,
    "system": false,
    "type": "number"
  }))

  // update field
  collection.fields.addAt(7, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text909375167",
    "max": 0,
    "min": 0,
    "name": "minimum_running_time",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  // update field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "date4272768884",
    "max": "",
    "min": "",
    "name": "first_execution",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "date"
  }))

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
    "required": true,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // update field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "number4263700706",
    "max": null,
    "min": null,
    "name": "total_runs",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // update field
  collection.fields.addAt(7, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text909375167",
    "max": 0,
    "min": 0,
    "name": "minimum_running_time",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // update field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "date4272768884",
    "max": "",
    "min": "",
    "name": "first_execution",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "date"
  }))

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
