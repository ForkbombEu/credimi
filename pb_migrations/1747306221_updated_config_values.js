/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2784107861")

  // remove field
  collection.fields.removeById("relation518144557")

  // remove field
  collection.fields.removeById("relation2462348188")

  // add field
  collection.fields.addAt(5, new Field({
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

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2784107861")

  // add field
  collection.fields.addAt(5, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2923319880",
    "hidden": false,
    "id": "relation518144557",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "test_suite",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(6, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_863811952",
    "hidden": false,
    "id": "relation2462348188",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "provider",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // remove field
  collection.fields.removeById("relation3479234172")

  return app.save(collection)
})
