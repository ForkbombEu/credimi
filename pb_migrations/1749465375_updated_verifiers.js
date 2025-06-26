/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_131690875")

  // add field
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1579384326",
    "max": 0,
    "min": 3,
    "name": "name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "file3834550803",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": [
      "image/png",
      "image/jpeg",
      "image/gif",
      "image/webp",
      "image/svg+xml"
    ],
    "name": "logo",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  // add field
  collection.fields.addAt(4, new Field({
    "exceptDomains": [],
    "hidden": false,
    "id": "url4101391790",
    "name": "url",
    "onlyDomains": [],
    "presentable": false,
    "required": true,
    "system": false,
    "type": "url"
  }))

  // add field
  collection.fields.addAt(5, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "url3156968278",
    "name": "repository_url",
    "onlyDomains": null,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "url"
  }))

  // add field
  collection.fields.addAt(6, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text2833051024",
    "max": 0,
    "min": 0,
    "name": "standard_and_version",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "select3736761055",
    "maxSelect": 3,
    "name": "format",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "SD-JWT",
      "mDOC",
      "W3C-VC"
    ]
  }))

  // add field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "select384687787",
    "maxSelect": 2,
    "name": "signing_algorithms",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "ES256",
      "RSA"
    ]
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "select3281364177",
    "maxSelect": 2,
    "name": "cryptographic_binding_methods",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "jwk",
      "cose_key"
    ]
  }))

  // add field
  collection.fields.addAt(10, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1843675174",
    "max": 99999999999999,
    "min": 0,
    "name": "description",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(11, new Field({
    "hidden": false,
    "id": "json1214815598",
    "maxSize": 0,
    "name": "conformance_checks",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_131690875")

  // remove field
  collection.fields.removeById("text1579384326")

  // remove field
  collection.fields.removeById("file3834550803")

  // remove field
  collection.fields.removeById("url4101391790")

  // remove field
  collection.fields.removeById("url3156968278")

  // remove field
  collection.fields.removeById("text2833051024")

  // remove field
  collection.fields.removeById("select3736761055")

  // remove field
  collection.fields.removeById("select384687787")

  // remove field
  collection.fields.removeById("select3281364177")

  // remove field
  collection.fields.removeById("text1843675174")

  // remove field
  collection.fields.removeById("json1214815598")

  return app.save(collection)
})
