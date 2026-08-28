/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  collection.fields.addAt(15, new Field({
    "hidden": false,
    "id": "file_fcaf_report_pdf",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": ["application/pdf"],
    "name": "fcaf_report_pdf",
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
  collection.fields.removeById("file_fcaf_report_pdf")
  return app.save(collection)
})
