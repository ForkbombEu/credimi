/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // remove field
  collection.fields.removeById("file325763347")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // add field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "file325763347",
    "maxSelect": 1,
    "maxSize": 524288000,
    "mimeTypes": [
      "video/mpeg",
      "video/mp4",
      "video/webm"
    ],
    "name": "result",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
})
