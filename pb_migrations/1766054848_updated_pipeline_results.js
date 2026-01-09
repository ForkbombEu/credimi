/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "file2051935673",
    "maxSelect": 99,
    "maxSize": 524288000,
    "mimeTypes": [],
    "name": "video_results",
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
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "file2051935673",
    "maxSelect": 99,
    "maxSize": 524288000,
    "mimeTypes": [
      "video/mpeg",
      "video/mp4",
      "video/webm"
    ],
    "name": "video_results",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
})
