/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3577178630")

  // only server code issues or modifies keys; owners may list, view and revoke their own
  unmarshal({
    "createRule": null,
    "deleteRule": "user = @request.auth.id",
    "listRule": "user = @request.auth.id",
    "updateRule": null,
    "viewRule": "user = @request.auth.id"
  }, collection)

  collection.fields.getByName("key").hidden = true

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3577178630")

  unmarshal({
    "createRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id",
    "deleteRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id",
    "listRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id",
    "updateRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id",
    "viewRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id"
  }, collection)

  collection.fields.getByName("key").hidden = false

  return app.save(collection)
})
