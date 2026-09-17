/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "number1789500001",
    "max": null,
    "min": null,
    "name": "minimum_running_time_seconds",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  collection.fields.removeById("number1789500001")

  return app.save(collection)
})
