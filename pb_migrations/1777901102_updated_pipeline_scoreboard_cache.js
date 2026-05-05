/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // add field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "number3689223975",
    "max": null,
    "min": null,
    "name": "CI_runs",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // remove field
  collection.fields.removeById("number3689223975")

  return app.save(collection)
})
