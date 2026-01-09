/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "listRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)",
    "viewRule": "published = true || (\n  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id\n)"
  }, collection)

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "bool1748787223",
    "name": "published",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1403167086")

  // update collection data
  unmarshal({
    "listRule": "",
    "viewRule": ""
  }, collection)

  // remove field
  collection.fields.removeById("bool1748787223")

  return app.save(collection)
})
