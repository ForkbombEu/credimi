/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_xD8UMaxUgN` ON `pipeline_scoreboard_cache` (`pipeline`)"
    ],
    "name": "pipeline_scoreboard_cache"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_xD8UMaxUgN` ON `pipeline_results_aggegrates` (`pipeline`)"
    ],
    "name": "pipeline_results_aggegrates"
  }, collection)

  return app.save(collection)
})
