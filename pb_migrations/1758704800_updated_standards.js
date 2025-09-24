/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_216647879")

  // remove field
  collection.fields.removeById("relation2818242420")

  // remove field
  collection.fields.removeById("relation1590653244")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_216647879")

  // add field
  collection.fields.addAt(7, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_216647879",
    "hidden": false,
    "id": "relation2818242420",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "siblings",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2923319880",
    "hidden": false,
    "id": "relation1590653244",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "test_suites",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
})
