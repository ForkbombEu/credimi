/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2450633325")

  // remove field
  collection.fields.removeById("relation3479234172")

  // remove field
  collection.fields.removeById("relation3586217458")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2450633325")

  // add field
  collection.fields.addAt(1, new Field({
    "cascadeDelete": false,
    "collectionId": "aako88kt3br4npt",
    "hidden": false,
    "id": "relation3479234172",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "owner",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(2, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_3643163317",
    "hidden": false,
    "id": "relation3586217458",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "conformance_check",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
})
