/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2201295156")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_jjqeX0KhkF` ON `wallet_versions` (\n  `wallet`,\n  `canonified_tag`,\n  `owner`\n)"
    ]
  }, collection)

  // add field
  collection.fields.addAt(4, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text2621739057",
    "max": 0,
    "min": 0,
    "name": "canonified_tag",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2201295156")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_jjqeX0KhkF` ON `wallet_versions` (\n  `wallet`,\n  `tag`,\n  `owner`\n)"
    ]
  }, collection)

  // remove field
  collection.fields.removeById("text2621739057")

  return app.save(collection)
})
