/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_678514665")

  // update collection data
  unmarshal({
    "deleteRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\"",
    "listRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)",
    "updateRule": "@request.body.owner:isset = false &&\n@request.body.url:isset = false && (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id &&\n  @collection.orgAuthorizations.role.name ?= \"owner\"\n)",
    "viewRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_678514665")

  // update collection data
  unmarshal({
    "deleteRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= id &&\n@collection.orgAuthorizations.role.name ?= \"owner\"",
    "listRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= id\n)",
    "updateRule": "@request.body.owner:isset = false &&\n@request.body.url:isset = false && (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= id &&\n  @collection.orgAuthorizations.role.name ?= \"owner\"\n)",
    "viewRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= id\n)"
  }, collection)

  return app.save(collection)
})
