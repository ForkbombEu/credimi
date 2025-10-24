/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2405597765")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT id, name, canonified_name, logo\nFROM organizations\nWHERE id IN (\n    SELECT owner FROM wallets WHERE published = TRUE\n    UNION\n    SELECT owner FROM credential_issuers WHERE published = TRUE\n    UNION\n    SELECT owner FROM verifiers WHERE published = TRUE\n    UNION\n    SELECT owner FROM use_cases_verifications WHERE published = TRUE\n    UNION\n    SELECT owner FROM custom_checks WHERE published = TRUE\n);"
  }, collection)

  // remove field
  collection.fields.removeById("_clone_e0yH")

  // remove field
  collection.fields.removeById("_clone_PWA8")

  // remove field
  collection.fields.removeById("_clone_E2nM")

  // add field
  collection.fields.addAt(1, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_wssX",
    "max": 0,
    "min": 2,
    "name": "name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_21kg",
    "max": 0,
    "min": 0,
    "name": "canonified_name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "_clone_thtH",
    "maxSelect": 1,
    "maxSize": 5242880,
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
    "thumbs": null,
    "type": "file"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2405597765")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT id, name, canonified_name, logo\nFROM organizations\nWHERE id IN (\n    SELECT owner FROM wallets WHERE published = TRUE\n    UNION\n    SELECT owner FROM credential_issuers WHERE published = TRUE\n    UNION\n    SELECT owner FROM credentials WHERE published = TRUE\n    UNION\n    SELECT owner FROM verifiers WHERE published = TRUE\n    UNION\n    SELECT owner FROM use_cases_verifications WHERE published = TRUE\n    UNION\n     SELECT owner FROM custom_checks WHERE published = TRUE\n);"
  }, collection)

  // add field
  collection.fields.addAt(1, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_e0yH",
    "max": 0,
    "min": 2,
    "name": "name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_PWA8",
    "max": 0,
    "min": 0,
    "name": "canonified_name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "_clone_E2nM",
    "maxSelect": 1,
    "maxSize": 5242880,
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
    "thumbs": null,
    "type": "file"
  }))

  // remove field
  collection.fields.removeById("_clone_wssX")

  // remove field
  collection.fields.removeById("_clone_21kg")

  // remove field
  collection.fields.removeById("_clone_thtH")

  return app.save(collection)
})
