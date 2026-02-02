/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1427126806")

  // remove unique index on pipeline, owner
  unmarshal({
    "indexes": []
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1427126806")

  // restore unique index on pipeline, owner
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_NyrcunBOQR` ON `schedules` (\n  `pipeline`,\n  `owner`\n)"
    ]
  }, collection)

  return app.save(collection)
})
