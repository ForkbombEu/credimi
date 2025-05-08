/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT\n  w.id AS id,\n  w.name AS name,\n  w.description AS description,\n  w.logo AS avatar,\n  NULL AS avatar_url,\n  'wallets' AS type\nFROM wallets w\nWHERE w.name IS NOT NULL AND w.published = true"
  }, collection)

  // remove field
  collection.fields.removeById("_clone_z68g")

  // remove field
  collection.fields.removeById("_clone_ulrw")

  // remove field
  collection.fields.removeById("_clone_j0RX")

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

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT\n  w.id AS id,\n  w.name AS name,\n  w.description AS description,\n  w.logo AS avatar,\n  NULL AS avatar_url,\n  'wallets' AS type\nFROM wallets w\nWHERE w.name IS NOT NULL"
  }, collection)

  // add field
  collection.fields.addAt(1, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_z68g",
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
    "id": "_clone_ulrw",
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
    "id": "_clone_j0RX",
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

  // remove field
  collection.fields.removeById("_clone_z91V")

  // remove field
  collection.fields.removeById("_clone_VbM2")

  // remove field
  collection.fields.removeById("_clone_YUPX")

  return app.save(collection)
})
