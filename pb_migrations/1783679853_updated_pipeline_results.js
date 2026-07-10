/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "file1855231354",
    "maxSelect": 99,
    "maxSize": 0,
    "mimeTypes": [],
    "name": "maestro_screenshots",
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
  collection.fields.removeById("file1855231354")

  return app.save(collection)
})
