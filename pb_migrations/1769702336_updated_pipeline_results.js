/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "file3776154291",
    "maxSelect": 99,
    "maxSize": 0,
    "mimeTypes": [],
    "name": "logcats",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // remove field
  collection.fields.removeById("file3776154291")

  return app.save(collection)
})
