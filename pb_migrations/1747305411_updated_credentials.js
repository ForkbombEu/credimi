/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_183765882")

  // update collection data
  unmarshal({
    "deleteRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= credential_issuer.owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\"",
    "listRule": "(published = true && credential_issuer.published = true) || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= credential_issuer.owner.id\n)",
    "updateRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= credential_issuer.owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\"",
    "viewRule": "(published = true && credential_issuer.published = true) || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= credential_issuer.owner.id\n)"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_183765882")

  // update collection data
  unmarshal({
    "deleteRule": null,
    "listRule": "(published = true && credential_issuer.published = true) ||\n@request.auth.id = credential_issuer.owner.id",
    "updateRule": "@request.auth.id = credential_issuer.owner.id",
    "viewRule": "(published = true && credential_issuer.published = true) ||\n@request.auth.id = credential_issuer.owner.id"
  }, collection)

  return app.save(collection)
})
