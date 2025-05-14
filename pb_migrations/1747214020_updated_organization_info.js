/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_863811952")

  // update collection data
  unmarshal({
    "deleteRule": "",
    "indexes": [],
    "updateRule": ""
  }, collection)

  // remove field
  collection.fields.removeById("relation3479234172")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_863811952")

  // update collection data
  unmarshal({
    "deleteRule": "@request.auth.id = owner.id",
    "indexes": [
      "CREATE INDEX `idx_qmfBsdc8jj` ON `organization_info` (`owner`)"
    ],
    "updateRule": "@request.auth.id = owner.id"
  }, collection)

  // add field
  collection.fields.addAt(8, new Field({
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
