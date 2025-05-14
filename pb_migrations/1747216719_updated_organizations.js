/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  // remove field
  collection.fields.removeById("zhuxbrib")

  // remove field
  collection.fields.removeById("pjjpq1r4")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  // add field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "zhuxbrib",
    "maxSelect": 1,
    "maxSize": 5242880,
    "mimeTypes": [
      "image/png",
      "image/jpeg",
      "image/webp",
      "image/svg+xml"
    ],
    "name": "logo",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": null,
    "type": "file"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "convertURLs": false,
    "hidden": false,
    "id": "pjjpq1r4",
    "maxSize": 0,
    "name": "description",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "editor"
  }))

  return app.save(collection)
})
