/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  // add field
  collection.fields.addAt(999, new Field({
    "hidden": false,
    "id": "number1770216148",
    "max": null,
    "min": null,
    "name": "max_pipelines_in_queue",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  // remove field
  collection.fields.removeById("number1770216148")

  return app.save(collection)
})
