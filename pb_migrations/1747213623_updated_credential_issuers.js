/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_678514665")

  // update collection data
  unmarshal({
    "deleteRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner_org.id &&\n@collection.orgAuthorizations.role.name = \"owner\"",
    "listRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner_org.id\n)",
    "updateRule": "@request.body.owner:isset = false &&\n@request.body.url:isset = false && (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner_org.id &&\n  @collection.orgAuthorizations.role.name = \"owner\"\n)",
    "viewRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner_org.id\n)"
  }, collection)

  // remove field
  collection.fields.removeById("relation3479234172")

  // add field
  collection.fields.addAt(8, new Field({
    "cascadeDelete": false,
    "collectionId": "aako88kt3br4npt",
    "hidden": false,
    "id": "relation3722355914",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "owner_org",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_678514665")

  // update collection data
  unmarshal({
    "deleteRule": "owner.id = @request.auth.id",
    "listRule": "published = true || owner.id = @request.auth.id",
    "updateRule": "@request.body.owner:isset = false &&\n@request.body.url:isset = false",
    "viewRule": "published = true || owner.id = @request.auth.id"
  }, collection)

  // add field
  collection.fields.addAt(2, new Field({
    "cascadeDelete": false,
    "collectionId": "_pb_users_auth_",
    "hidden": false,
    "id": "relation3479234172",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "owner",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // remove field
  collection.fields.removeById("relation3722355914")

  return app.save(collection)
})
