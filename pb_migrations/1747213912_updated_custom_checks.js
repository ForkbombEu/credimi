/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // update collection data
  unmarshal({
    "deleteRule": "",
    "listRule": "public = true",
    "updateRule": "",
    "viewRule": "public = true"
  }, collection)

  // remove field
  collection.fields.removeById("relation3479234172")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // update collection data
  unmarshal({
    "deleteRule": "owner.id = @request.auth.id",
    "listRule": "owner.id = @request.auth.id || public = true",
    "updateRule": "owner.id = @request.auth.id",
    "viewRule": "owner.id = @request.auth.id || public = true"
  }, collection)

  // add field
  collection.fields.addAt(5, new Field({
    "cascadeDelete": false,
    "collectionId": "_pb_users_auth_",
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
})
