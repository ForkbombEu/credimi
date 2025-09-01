/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3577178630")

  // update collection data
  unmarshal({
    "createRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id",
    "deleteRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id",
    "listRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id",
    "updateRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id",
    "viewRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3577178630")

  // update collection data
  unmarshal({
    "createRule": null,
    "deleteRule": null,
    "listRule": null,
    "updateRule": null,
    "viewRule": null
  }, collection)

  return app.save(collection)
})
