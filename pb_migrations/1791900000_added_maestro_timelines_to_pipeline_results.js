/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // add field
  collection.fields.add(new Field({
    "hidden": false,
    "id": "file_maestro_timelines",
    "maxSelect": 99,
    "maxSize": 52428800,
    "mimeTypes": [],
    "name": "maestro_timelines",
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
  collection.fields.removeById("file_maestro_timelines")

  return app.save(collection)
})
