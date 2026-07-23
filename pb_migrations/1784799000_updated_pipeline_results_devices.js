// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")

  collection.fields.add(new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_3594974968",
    "hidden": false,
    "id": "relation1784799000",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "devices",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441")
  collection.fields.removeById("relation1784799000")
  return app.save(collection)
})
