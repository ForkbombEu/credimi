// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // remove field
  collection.fields.removeById("editor3929123204")

  // add field
  collection.fields.addAt(12, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3929123204",
    "max": 0,
    "min": 0,
    "name": "yaml",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1108732172")

  // add field
  collection.fields.addAt(4, new Field({
    "convertURLs": false,
    "hidden": false,
    "id": "editor3929123204",
    "maxSize": 0,
    "name": "yaml",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "editor"
  }))

  // remove field
  collection.fields.removeById("text3929123204")

  return app.save(collection)
})
