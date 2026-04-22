/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // add field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "number1956432625",
    "max": null,
    "min": null,
    "name": "success_rate",
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
  collection.fields.removeById("number1956432625")

  return app.save(collection)
})
