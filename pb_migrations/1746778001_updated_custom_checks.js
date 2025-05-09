/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // update collection data
  unmarshal({
    "createRule": "@request.auth.id != \"\"",
    "deleteRule": "owner.id = @request.auth.id",
    "listRule": "owner.id = @request.auth.id || public = true",
    "viewRule": "owner.id = @request.auth.id || public = true"
  }, collection)

  // remove field
  collection.fields.removeById("select2044034195")

  // remove field
  collection.fields.removeById("relation284678023")

  // remove field
  collection.fields.removeById("json1874629670")

  // remove field
  collection.fields.removeById("text1542800728")

  // add field
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text284678023",
    "max": 0,
    "min": 0,
    "name": "standard",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(5, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "url3408188910",
    "name": "homepage",
    "onlyDomains": null,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "url"
  }))

  // add field
  collection.fields.addAt(6, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "url1560172493",
    "name": "repository",
    "onlyDomains": null,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "url"
  }))

  // update field
  collection.fields.addAt(3, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3206337475",
    "max": 25,
    "min": 0,
    "name": "standard_version",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // update field
  collection.fields.addAt(11, new Field({
    "hidden": false,
    "id": "bool4208731335",
    "name": "public",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // update collection data
  unmarshal({
    "createRule": "",
    "deleteRule": null,
    "listRule": "",
    "viewRule": ""
  }, collection)

  // add field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "select2044034195",
    "maxSelect": 1,
    "name": "tooling",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "stepci"
    ]
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_216647879",
    "hidden": false,
    "id": "relation284678023",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "standard",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(10, new Field({
    "hidden": false,
    "id": "json1874629670",
    "maxSize": 0,
    "name": "tags",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  // add field
  collection.fields.addAt(11, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1542800728",
    "max": 0,
    "min": 0,
    "name": "field",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // remove field
  collection.fields.removeById("text284678023")

  // remove field
  collection.fields.removeById("url3408188910")

  // remove field
  collection.fields.removeById("url1560172493")

  // update field
  collection.fields.addAt(8, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3206337475",
    "max": 25,
    "min": 0,
    "name": "version",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // update field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "bool4208731335",
    "name": "isPublic",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
})
