/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3577178630")
  const superusersCollection = app.findCollectionByNameOrId("_superusers")

  // update user field to optional to support superuser-owned keys
  collection.fields.addAt(3, new Field({
    "cascadeDelete": false,
    "collectionId": "_pb_users_auth_",
    "hidden": false,
    "id": "relation2375276105",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "user",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  collection.fields.addAt(4, new Field({
    "cascadeDelete": false,
    "collectionId": superusersCollection.id,
    "hidden": false,
    "id": "relation4070824330",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "superuser",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select3180219609",
    "maxSelect": 1,
    "name": "key_type",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "user",
      "internal_admin"
    ]
  }))

  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "bool1647378504",
    "name": "revoked",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "date1864911767",
    "max": "",
    "min": "",
    "name": "expires_at",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "date"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3577178630")

  collection.fields.addAt(3, new Field({
    "cascadeDelete": false,
    "collectionId": "_pb_users_auth_",
    "hidden": false,
    "id": "relation2375276105",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "user",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "relation"
  }))

  collection.fields.removeById("relation4070824330")
  collection.fields.removeById("select3180219609")
  collection.fields.removeById("bool1647378504")
  collection.fields.removeById("date1864911767")

  return app.save(collection)
})
