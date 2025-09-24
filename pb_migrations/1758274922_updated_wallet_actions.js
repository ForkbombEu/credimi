/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_FV3PchKuqM` ON `wallet_actions` (\n  `owner`,\n  `uid`,\n  `wallet`\n)",
      "CREATE UNIQUE INDEX `idx_QSuTX94q9T` ON `wallet_actions` (`canonified_name`)"
    ]
  }, collection)

  // add field
  collection.fields.addAt(3, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text2077450625",
    "max": 0,
    "min": 0,
    "name": "canonified_name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_FV3PchKuqM` ON `wallet_actions` (\n  `owner`,\n  `uid`,\n  `wallet`\n)"
    ]
  }, collection)

  // remove field
  collection.fields.removeById("text2077450625")

  return app.save(collection)
})
