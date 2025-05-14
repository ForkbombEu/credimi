/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  // update collection data
  unmarshal({
    "createRule": null
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  // update collection data
  unmarshal({
    "createRule": "@request.auth.id != ''"
  }, collection)

  return app.save(collection)
})
