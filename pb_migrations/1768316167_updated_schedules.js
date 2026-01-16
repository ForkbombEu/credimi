/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1427126806")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_NyrcunBOQR` ON `schedules` (\n  `pipeline`,\n  `owner`\n)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1427126806")

  // update collection data
  unmarshal({
    "indexes": []
  }, collection)

  return app.save(collection)
})
