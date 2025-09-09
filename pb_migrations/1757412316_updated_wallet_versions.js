/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2201295156")

  // update collection data
  unmarshal({
    "createRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\" &&\n(\n  @request.body.android_installer:isset = true ||\n  @request.body.ios_installer:isset = true\n)"
  }, collection)

  // remove field
  collection.fields.removeById("select961728715")

  // add field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "file3111593885",
    "maxSelect": 1,
    "maxSize": 262144000,
    "mimeTypes": [],
    "name": "ios_installer",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "file2359244304",
    "maxSelect": 1,
    "maxSize": 262144000,
    "mimeTypes": [],
    "name": "android_installer",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2201295156")

  // update collection data
  unmarshal({
    "createRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\""
  }, collection)

  // add field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select961728715",
    "maxSelect": 1,
    "name": "platform",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "android",
      "ios"
    ]
  }))

  // remove field
  collection.fields.removeById("file3111593885")

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "file2359244304",
    "maxSelect": 1,
    "maxSize": 262144000,
    "mimeTypes": [],
    "name": "file",
    "presentable": false,
    "protected": false,
    "required": true,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
})
