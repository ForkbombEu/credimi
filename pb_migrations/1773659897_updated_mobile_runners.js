/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_500646217")

  // update field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "select2363381545",
    "maxSelect": 1,
    "name": "type",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "android_emulator",
      "redroid",
      "android_phone",
      "ios_simulator",
      "ios_phone"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_500646217")

  // update field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "select2363381545",
    "maxSelect": 1,
    "name": "type",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "android_emulator",
      "redroid",
      "android_phone",
      "ios_emulator",
      "ios_phone"
    ]
  }))

  return app.save(collection)
})
