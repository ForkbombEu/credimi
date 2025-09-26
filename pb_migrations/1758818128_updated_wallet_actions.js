/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_FV3PchKuqM` ON `wallet_actions` (`owner`, `uid`, `wallet`)",
      "CREATE UNIQUE INDEX `idx_QSuTX94q9T` ON `wallet_actions` (`wallet`,`canonified_name`)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_FV3PchKuqM` ON `wallet_actions` (`owner`, `uid`, `wallet`)",
      "CREATE UNIQUE INDEX `idx_QSuTX94q9T` ON `wallet_actions` (`canonified_name`)"
    ]
  }, collection)

  return app.save(collection)
})
