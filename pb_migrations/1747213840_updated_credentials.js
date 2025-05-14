/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_183765882")

  // update collection data
  unmarshal({
    "deleteRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name = \"owner\"",
    "listRule": "(published = true && credential_issuer.published = true) || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)",
    "updateRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name = \"owner\"",
    "viewRule": "(published = true && credential_issuer.published = true) || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)"
  }, collection)

  // add field
  collection.fields.addAt(13, new Field({
    "cascadeDelete": false,
    "collectionId": "aako88kt3br4npt",
    "hidden": false,
    "id": "relation3479234172",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "owner",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "relation"
  }))

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

  // remove field
  collection.fields.removeById("relation3479234172")

  return app.save(collection)
})
