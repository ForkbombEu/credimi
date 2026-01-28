/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1427126806")

  // add runner field
  collection.fields.addAt(5, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1234567890",
    "max": 0,
    "min": 0,
    "name": "runner",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // remove unique index on pipeline, owner
  unmarshal({
    "indexes": []
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1427126806")

  // remove runner field
  collection.fields.removeById("text1234567890")

  // restore unique index on pipeline, owner
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_NyrcunBOQR` ON `schedules` (\n  `pipeline`,\n  `owner`\n)"
    ]
  }, collection)

  return app.save(collection)
})
