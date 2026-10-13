/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_987692768")

  // superusers only publish news; published news stays publicly readable
  unmarshal({
    "createRule": null
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_987692768")

  unmarshal({
    "createRule": ""
  }, collection)

  return app.save(collection)
})
