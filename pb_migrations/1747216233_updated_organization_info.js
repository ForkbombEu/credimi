/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_863811952")

  // update field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "file3834550803",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": [
      "image/png",
      "image/jpeg",
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

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select1400097126",
    "maxSelect": 1,
    "name": "country",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "AL",
      "AD",
      "AM",
      "AT",
      "AZ",
      "BY",
      "BE",
      "BA",
      "BG",
      "HR",
      "CY",
      "CZ",
      "DK",
      "EE",
      "FI",
      "FR",
      "GE",
      "DE",
      "GR",
      "HU",
      "IS",
      "IE",
      "IT",
      "KZ",
      "XK",
      "LV",
      "LI",
      "LT",
      "LU",
      "MT",
      "MD",
      "MC",
      "ME",
      "NL",
      "MK",
      "NO",
      "PL",
      "PT",
      "RO",
      "RU",
      "SM",
      "RS",
      "SK",
      "SI",
      "ES",
      "SE",
      "CH",
      "TR",
      "UA",
      "GB",
      "VA",
      "Other"
    ]
  }))

  // update field
  collection.fields.addAt(5, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3793657363",
    "max": 0,
    "min": 0,
    "name": "legal_entity",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // update field
  collection.fields.addAt(6, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "url4106974746",
    "name": "external_website_url",
    "onlyDomains": null,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "url"
  }))

  // update field
  collection.fields.addAt(7, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "email3401084027",
    "name": "contact_email",
    "onlyDomains": null,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "email"
  }))

  // update field
  collection.fields.addAt(8, new Field({
    "cascadeDelete": false,
    "collectionId": "aako88kt3br4npt",
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

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_863811952")

  // update field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "file3834550803",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": [
      "image/png",
      "image/jpeg",
      "image/webp",
      "image/svg+xml"
    ],
    "name": "logo",
    "presentable": false,
    "protected": false,
    "required": true,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select1400097126",
    "maxSelect": 1,
    "name": "country",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "AL",
      "AD",
      "AM",
      "AT",
      "AZ",
      "BY",
      "BE",
      "BA",
      "BG",
      "HR",
      "CY",
      "CZ",
      "DK",
      "EE",
      "FI",
      "FR",
      "GE",
      "DE",
      "GR",
      "HU",
      "IS",
      "IE",
      "IT",
      "KZ",
      "XK",
      "LV",
      "LI",
      "LT",
      "LU",
      "MT",
      "MD",
      "MC",
      "ME",
      "NL",
      "MK",
      "NO",
      "PL",
      "PT",
      "RO",
      "RU",
      "SM",
      "RS",
      "SK",
      "SI",
      "ES",
      "SE",
      "CH",
      "TR",
      "UA",
      "GB",
      "VA",
      "Other"
    ]
  }))

  // update field
  collection.fields.addAt(5, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3793657363",
    "max": 0,
    "min": 0,
    "name": "legal_entity",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  // update field
  collection.fields.addAt(6, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "url4106974746",
    "name": "external_website_url",
    "onlyDomains": null,
    "presentable": false,
    "required": true,
    "system": false,
    "type": "url"
  }))

  // update field
  collection.fields.addAt(7, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "email3401084027",
    "name": "contact_email",
    "onlyDomains": null,
    "presentable": false,
    "required": true,
    "system": false,
    "type": "email"
  }))

  // update field
  collection.fields.addAt(8, new Field({
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

  return app.save(collection)
})
