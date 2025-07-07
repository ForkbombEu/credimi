/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_92944219")

  // update collection data
  unmarshal({
    "name": "verification_use_cases"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_92944219")

  // update collection data
  unmarshal({
    "name": "verification_use_case"
  }, collection)

  return app.save(collection)
})
