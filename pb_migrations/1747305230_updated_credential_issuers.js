/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_678514665")

  // update collection data
  unmarshal({
    "deleteRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= id &&\n@collection.orgAuthorizations.role.name ?= \"owner\"",
    "listRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= id\n)",
    "updateRule": "@request.body.owner:isset = false &&\n@request.body.url:isset = false && (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= id &&\n  @collection.orgAuthorizations.role.name ?= \"owner\"\n)",
    "viewRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= id\n)"
  }, collection)

  // add field
  collection.fields.addAt(1, new Field({
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

  // update field
  collection.fields.addAt(2, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "url4101391790",
    "name": "url",
    "onlyDomains": null,
    "presentable": false,
    "required": true,
    "system": false,
    "type": "url"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_678514665")

  // update collection data
  unmarshal({
    "deleteRule": "",
    "listRule": "published = true",
    "updateRule": "@request.body.owner:isset = false &&\n@request.body.url:isset = false",
    "viewRule": "published = true"
  }, collection)

  // remove field
  collection.fields.removeById("relation3479234172")

  // update field
  collection.fields.addAt(1, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "url4101391790",
    "name": "url",
    "onlyDomains": null,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "url"
  }))

  return app.save(collection)
})
