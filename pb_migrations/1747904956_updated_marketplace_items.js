/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT\n  item.id,\n  item.type,\n  item.name,\n  item.description,\n  item.updated,\n  item.avatar,\n  -- item.avatar_url,\n  item.organization_id,\n  o.name AS organization_name\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    NULL AS avatar,\n    -- 'https://demo.credimi.io' || '/api/files/' || (SELECT id FROM _collections WHERE name = 'wallets') || '/' || w.id || '/' || w.logo AS avatar_url,\n    'wallets' AS type,\n    w.owner AS organization_id\n  FROM wallets w\n  WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION ALL\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    -- ci.logo_url AS avatar_url,\n    'credential_issuers' AS type,\n    ci.owner AS organization_id\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n\n  UNION ALL\n\n  SELECT\n    cr.id AS id,\n    cr.name AS name,\n    NULL AS description,\n    cr.updated AS updated,\n    NULL AS avatar,\n    -- cr.logo AS avatar_url,\n    'credentials' AS type,\n    cr.owner AS organization_id\n  FROM credentials cr\n  WHERE cr.name IS NOT NULL AND cr.published = true\n\n  UNION ALL\n\n  SELECT\n    cc.id AS id,\n    cc.name AS name,\n    cc.description AS description,\n    cc.updated AS updated,\n    NULL AS avatar,\n    -- 'https://demo.credimi.io' || '/api/files/' || (SELECT id FROM _collections WHERE name = 'custom_checks') || '/' || cc.id || '/' || cc.logo AS avatar_url, \n    'custom_checks' AS type, \n    cc.owner AS organization_id\n  FROM custom_checks cc\n  WHERE cc.name IS NOT NULL AND cc.public = true \n) AS item\n\nLEFT JOIN organizations o ON item.organization_id = o.id;"
  }, collection)

  // remove field
  collection.fields.removeById("json2190673250")

  // remove field
  collection.fields.removeById("_clone_IpAB")

  // add field
  collection.fields.addAt(7, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_AXHS",
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
    "viewQuery": "SELECT\n  item.id,\n  item.type,\n  item.name,\n  item.description,\n  item.updated,\n  item.avatar,\n  item.avatar_url,\n  item.organization_id,\n  o.name AS organization_name\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    NULL AS avatar,\n    'https://demo.credimi.io' || '/api/files/' || (SELECT id FROM _collections WHERE name = 'wallets') || '/' || w.id || '/' || w.logo AS avatar_url,\n    'wallets' AS type,\n    w.owner AS organization_id\n  FROM wallets w\n  WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION ALL\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type,\n    ci.owner AS organization_id\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n\n  UNION ALL\n\n  SELECT\n    cr.id AS id,\n    cr.name AS name,\n    NULL AS description,\n    cr.updated AS updated,\n    NULL AS avatar,\n    cr.logo AS avatar_url,\n    'credentials' AS type,\n    cr.owner AS organization_id\n  FROM credentials cr\n  WHERE cr.name IS NOT NULL AND cr.published = true\n\n  UNION ALL\n\n  SELECT\n    cc.id AS id,\n    cc.name AS name,\n    cc.description AS description,\n    cc.updated AS updated,\n    NULL AS avatar,\n    'https://demo.credimi.io' || '/api/files/' || (SELECT id FROM _collections WHERE name = 'custom_checks') || '/' || cc.id || '/' || cc.logo AS avatar_url, \n    'custom_checks' AS type, \n    cc.owner AS organization_id\n  FROM custom_checks cc\n  WHERE cc.name IS NOT NULL AND cc.public = true \n) AS item\n\nLEFT JOIN organizations o ON item.organization_id = o.id;"
  }, collection)

  // add field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "json2190673250",
    "maxSize": 1,
    "name": "avatar_url",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  // add field
  collection.fields.addAt(8, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_IpAB",
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

  // remove field
  collection.fields.removeById("_clone_AXHS")

  return app.save(collection)
})
