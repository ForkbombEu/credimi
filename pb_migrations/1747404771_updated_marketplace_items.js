/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT \n  item.id, \n  item.type, \n  item.name, \n  item.description, \n  item.updated, \n  item.avatar, \n  item.avatar_url, \n  item.organization_id,\n  o.name AS organization_name\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    w.logo AS avatar,\n    NULL AS avatar_url,\n    'wallets' AS type,\n    w.owner AS organization_id\n  FROM wallets w\n  WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION ALL\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type,\n    ci.owner AS organization_id\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n\n  UNION ALL\n\n  SELECT\n    cr.id AS id,\n    cr.name AS name,\n    NULL AS description,\n    cr.updated AS updated,\n    NULL AS avatar,\n    NULL AS avatar_url,\n    'credentials' AS type,\n    cr.owner AS organization_id\n  FROM credentials cr\n  WHERE cr.name IS NOT NULL AND cr.published = true\n) AS item\n\nLEFT JOIN organizations o ON item.organization_id = o.id"
  }, collection)

  // add field
  collection.fields.addAt(8, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_sDM8",
    "max": 0,
    "min": 2,
    "name": "organization_name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT id, type, name, description, updated, avatar, avatar_url, organization_id\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    w.logo AS avatar,\n    NULL AS avatar_url,\n    'wallets' AS type,\n    w.owner AS organization_id\n  FROM wallets w\n  WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type,\n    ci.owner AS organization_id\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n\n  UNION\n\n  SELECT\n    cr.id AS id,\n    cr.name AS name,\n    NULL as description,\n    cr.updated AS updated,\n    NULL as avatar,\n    NULL as avatar_url,\n    'credentials' AS type,\n    cr.owner AS organization_id\n  FROM credentials as cr\n  WHERE cr.name IS NOT NULL AND cr.published = true\n)"
  }, collection)

  // remove field
  collection.fields.removeById("_clone_sDM8")

  return app.save(collection)
})
