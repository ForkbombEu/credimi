/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3918024567")

  // a subscription stays bound to its owner: the body cannot reassign user
  unmarshal({
    "updateRule": "user = @request.auth.id && (@request.body.user:isset = false || @request.body.user = @request.auth.id)"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3918024567")

  unmarshal({
    "updateRule": "user = @request.auth.id"
  }, collection)

  return app.save(collection)
})
