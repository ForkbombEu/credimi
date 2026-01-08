/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_jfofwpkXdf` ON `pipeline_results` (\n  `pipeline`,\n  `canonified_identifier`\n)"
    ]
  }, collection)

  // add field
  collection.fields.addAt(7, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text785069557",
    "max": 0,
    "min": 0,
    "name": "canonified_identifier",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_jfofwpkXdf` ON `pipeline_results` (\n  `owner`,\n  `pipeline`\n)",
      "CREATE UNIQUE INDEX `idx_QE0nzfgj8n` ON `pipeline_results` (\n  `owner`,\n  `workflow_id`,\n  `run_id`\n)"
    ]
  }, collection)

  // remove field
  collection.fields.removeById("text785069557")

  return app.save(collection)
})
