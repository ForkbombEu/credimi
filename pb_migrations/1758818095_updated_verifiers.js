/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_131690875")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_3mgbTTqvSj` ON `verifiers` (`owner`, `canonified_name`)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_131690875")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_3mgbTTqvSj` ON `verifiers` (`canonified_name`)"
    ]
  }, collection)

  return app.save(collection)
})
