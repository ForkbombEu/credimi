/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_xD8UMaxUgN` ON `pipeline_results_aggegrates` (`pipeline`)"
    ],
    "name": "pipeline_results_aggegrates"
  }, collection)

  // add field
  collection.fields.addAt(16, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_120182150",
    "hidden": false,
    "id": "relation2524621420",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "wallets",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(17, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_131690875",
    "hidden": false,
    "id": "relation1676842979",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "verifiers",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(18, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_678514665",
    "hidden": false,
    "id": "relation1987989935",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "issuers",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_xD8UMaxUgN` ON `pipeline_results_aggegrate` (`pipeline`)"
    ],
    "name": "pipeline_results_aggegrate"
  }, collection)

  // remove field
  collection.fields.removeById("relation2524621420")

  // remove field
  collection.fields.removeById("relation1676842979")

  // remove field
  collection.fields.removeById("relation1987989935")

  return app.save(collection)
})
