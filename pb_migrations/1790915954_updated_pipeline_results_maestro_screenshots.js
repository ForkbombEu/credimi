/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  // Every mobile step of a run appends its Maestro screenshots to the run
  // record; a complete FCAF validation stores several hundred of them.
  collection.fields.getByName("maestro_screenshots").maxSelect = 2000

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  collection.fields.getByName("maestro_screenshots").maxSelect = 99

  return app.save(collection)
})
