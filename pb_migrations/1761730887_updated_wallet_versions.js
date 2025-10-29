/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2201295156")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "file2359244304",
    "maxSelect": 1,
    "maxSize": 425000000,
    "mimeTypes": [],
    "name": "android_installer",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2201295156")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "file2359244304",
    "maxSelect": 1,
    "maxSize": 262144000,
    "mimeTypes": [],
    "name": "android_installer",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
})
