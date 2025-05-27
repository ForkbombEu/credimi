/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("_pb_users_auth_")

  // add field
  collection.fields.addAt(9, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3463958721",
    "max": 0,
    "min": 0,
    "name": "Timezone",
    "pattern": "^[A-Za-z]+(?:[._-][A-Za-z0-9+-]+)*\\/[A-Za-z0-9._+-]+$",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("_pb_users_auth_")

  // remove field
  collection.fields.removeById("text3463958721")

  return app.save(collection)
})
