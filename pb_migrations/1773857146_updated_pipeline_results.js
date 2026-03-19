/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // update field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "file1654512270",
    "maxSelect": 99,
    "maxSize": 524288000,
    "mimeTypes": [],
    "name": "ios_logstreams",
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

  // update field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "file1654512270",
    "maxSelect": 99,
    "maxSize": 0,
    "mimeTypes": [],
    "name": "ios_logstreams",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
})
