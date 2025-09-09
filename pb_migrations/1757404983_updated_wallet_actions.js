/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_FV3PchKuqM` ON `wallet_actions` (\n  `owner`,\n  `uid`,\n  `wallet`\n)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "indexes": []
  }, collection)

  return app.save(collection)
})
