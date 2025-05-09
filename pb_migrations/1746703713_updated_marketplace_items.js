/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT id, type, name, description, updated, avatar, avatar_url\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    w.logo AS avatar,\n    NULL AS avatar_url,\n    'wallets' AS type\n  FROM wallets w WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n)"
  }, collection)

  // add field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "json3332085495",
    "maxSize": 1,
    "name": "updated",
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
    "viewQuery": "SELECT id, type, name, description, avatar, avatar_url\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.logo AS avatar,\n    NULL AS avatar_url,\n    'wallets' AS type\n  FROM wallets w WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n)"
  }, collection)

  // remove field
  collection.fields.removeById("json3332085495")

  return app.save(collection)
})
