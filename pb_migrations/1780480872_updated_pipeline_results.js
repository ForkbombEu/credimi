/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // add field
  collection.fields.addAt(11, new Field({
    "hidden": false,
    "id": "json278732222",
    "maxSize": 0,
    "name": "credential_well_knowns",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  // add field
  collection.fields.addAt(12, new Field({
    "hidden": false,
    "id": "json47227981",
    "maxSize": 0,
    "name": "presentation_results",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  // add field
  collection.fields.addAt(13, new Field({
    "hidden": false,
    "id": "file3291445124",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": [],
    "name": "report",
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
  collection.fields.removeById("json278732222")

  // remove field
  collection.fields.removeById("json47227981")

  // remove field
  collection.fields.removeById("file3291445124")

  return app.save(collection)
})
