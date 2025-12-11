/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_jfofwpkXdf` ON `pipeline_results` (\n  `canonified_identifier`,\n  `owner`\n)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_jfofwpkXdf` ON `pipeline_results` (\n  `pipeline`,\n  `canonified_identifier`\n)"
    ]
  }, collection)

  return app.save(collection)
})
