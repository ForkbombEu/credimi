/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "name": "marketplace_items",
    "viewQuery": "SELECT id, name, description, avatar, avatar_url\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.logo AS avatar,\n    NULL AS avatar_url,\n    'wallets' AS type\n  FROM wallets w WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n)"
  }, collection)

  // remove field
  collection.fields.removeById("_clone_z91V")

  // remove field
  collection.fields.removeById("_clone_VbM2")

  // remove field
  collection.fields.removeById("_clone_YUPX")

  // remove field
  collection.fields.removeById("json2363381545")

  // add field
  collection.fields.addAt(1, new Field({
    "hidden": false,
    "id": "json1579384326",
    "maxSize": 1,
    "name": "name",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  // add field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "json1843675174",
    "maxSize": 1,
    "name": "description",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "json376926767",
    "maxSize": 1,
    "name": "avatar",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "name": "marketplace_entries",
    "viewQuery": "SELECT\n  w.id AS id,\n  w.name AS name,\n  w.description AS description,\n  w.logo AS avatar,\n  NULL AS avatar_url,\n  'wallets' AS type\nFROM wallets w\nWHERE w.name IS NOT NULL AND w.published = true"
  }, collection)

  // add field
  collection.fields.addAt(1, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_z91V",
    "max": 0,
    "min": 0,
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
    "convertURLs": false,
    "hidden": false,
    "id": "_clone_VbM2",
    "maxSize": 0,
    "name": "description",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "editor"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "_clone_YUPX",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": [],
    "name": "avatar",
    "presentable": false,
    "protected": false,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  // add field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "json2363381545",
    "maxSize": 1,
    "name": "type",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  // remove field
  collection.fields.removeById("json1579384326")

  // remove field
  collection.fields.removeById("json1843675174")

  // remove field
  collection.fields.removeById("json376926767")

  return app.save(collection)
})
