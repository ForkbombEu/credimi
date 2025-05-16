/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT id, type, name, description, updated, avatar, avatar_url, organization_id\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    w.logo AS avatar,\n    NULL AS avatar_url,\n    'wallets' AS type,\n    w.owner AS organization_id\n  FROM wallets w\n  WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type,\n    ci.owner AS organization_id\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n\n  UNION\n\n  SELECT\n    cr.id AS id,\n    cr.name AS name,\n    NULL as description,\n    cr.updated AS updated,\n    NULL as avatar,\n    NULL as avatar_url,\n    'credentials' AS type,\n    cr.owner AS organization_id\n  FROM credentials as cr\n  WHERE cr.name IS NOT NULL AND cr.published = true\n)"
  }, collection)

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "json852009950",
    "maxSize": 1,
    "name": "organization_id",
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
    "viewQuery": "SELECT id, type, name, description, updated, avatar, avatar_url\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    w.logo AS avatar,\n    NULL AS avatar_url,\n    'wallets' AS type\n  FROM wallets w WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n)"
  }, collection)

  // remove field
  collection.fields.removeById("json852009950")

  return app.save(collection)
})
