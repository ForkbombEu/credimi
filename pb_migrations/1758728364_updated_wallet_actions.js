/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_FV3PchKuqM` ON `wallet_actions` (\n  `owner`,\n  `name`,\n  `wallet`\n)"
    ]
  }, collection)

  // remove field
  collection.fields.removeById("text1402668550")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_FV3PchKuqM` ON `wallet_actions` (\n  `owner`,\n  `uid`,\n  `wallet`\n)"
    ]
  }, collection)

  // add field
  collection.fields.addAt(3, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1402668550",
    "max": 0,
    "min": 3,
    "name": "uid",
    "pattern": "^[a-z][a-z0-9_-]*$",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
})
