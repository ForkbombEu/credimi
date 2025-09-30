/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_92944219")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_hxtkHzU1Xk` ON `use_cases_verifications` (`verifier`,`canonified_name`)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_92944219")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_hxtkHzU1Xk` ON `use_cases_verifications` (`canonified_name`)"
    ]
  }, collection)

  return app.save(collection)
})
