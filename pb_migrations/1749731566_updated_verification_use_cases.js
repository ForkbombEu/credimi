/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_92944219")

  // update collection data
  unmarshal({
    "listRule": "(published = true && verifier.published = true) || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)",
    "viewRule": "(published = true && verifier.published = true) || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_92944219")

  // update collection data
  unmarshal({
    "listRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)",
    "viewRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)"
  }, collection)

  return app.save(collection)
})
