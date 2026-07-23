/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // remove field
  collection.fields.removeById("relation977024222")

  // add field
  collection.fields.addAt(2, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_3594974968",
    "hidden": false,
    "id": "relation4054680431",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "mobile_devices",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // add field
  collection.fields.addAt(2, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_500646217",
    "hidden": false,
    "id": "relation977024222",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "mobile_runners",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // remove field
  collection.fields.removeById("relation4054680431")

  return app.save(collection)
})
