/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  collection.fields.addAt(14, new Field({
    "hidden": false,
    "id": "file_fcaf_report",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": ["application/json"],
    "name": "fcaf_report",
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
  collection.fields.removeById("file_fcaf_report")
  return app.save(collection)
})
