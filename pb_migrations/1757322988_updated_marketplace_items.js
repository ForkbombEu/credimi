/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT\n  item.id,\n  item.type,\n  item.name,\n  item.description,\n  item.updated,\n  item.avatar,\n  item.avatar_url,\n  item.organization_id,\n  o.name AS organization_name\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    NULL AS avatar,\n    w.logo_url AS avatar_url,\n    'wallets' AS type,\n    w.owner AS organization_id\n  FROM wallets w\n  WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION ALL\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type,\n    ci.owner AS organization_id\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n\n  UNION ALL\n\n  SELECT\n    cr.id AS id,\n    COALESCE(NULLIF(cr.name, ''), cr.key) AS name,\n    NULL AS description,\n    cr.updated AS updated,\n    NULL AS avatar,\n    cr.logo_data AS avatar_url,\n    'credentials' AS type,\n    cr.owner AS organization_id\n  FROM credentials cr\n  JOIN credential_issuers AS i ON cr.credential_issuer = i.id\n  WHERE cr.name IS NOT NULL AND cr.published = true AND i.published = true\n\n  UNION ALL\n\n  SELECT\n    cc.id AS id,\n    cc.name AS name,\n    cc.description AS description,\n    cc.updated AS updated,\n    JSON_OBJECT(\n        'id', cc.id,\n        'collectionId', (SELECT id FROM _collections WHERE name = 'custom_checks'), \n        'collectionName', 'custom_checks', \n        'image_file', cc.logo\n    ) AS avatar,\n    NULL AS avatar_url,\n    'custom_checks' AS type, \n    cc.owner AS organization_id\n  FROM custom_checks cc\n  WHERE cc.name IS NOT NULL AND cc.public = true\n\n  UNION ALL\n\n  SELECT\n    vr.id AS id,\n    vr.name AS name,\n    vr.description AS description,\n    vr.updated AS updated,\n    JSON_OBJECT(\n      'id', vr.id,\n      'collectionId', (SELECT id FROM _collections WHERE name = 'verifiers'), \n      'collectionName', 'verifiers', \n      'image_file', vr.logo\n    ) AS avatar,\n    NULL AS avatar_url,\n    'verifiers' AS type, \n    vr.owner AS organization_id\n  FROM verifiers as vr\n  WHERE vr.name IS NOT NULL AND vr.published = true\n\n  UNION ALL\n\n  SELECT\n    vuc.id AS id,\n    vuc.name AS name,\n    vuc.description AS description,\n    vuc.updated as updated,\n    NULL as avatar,\n    NULL as avatar_url,\n    'use_cases_verifications' as type,\n    vuc.owner as organization_id\n  FROM use_cases_verifications as vuc\n  JOIN verifiers as v ON vuc.verifier = v.id\n  WHERE vuc.published = true AND v.published = true\n) AS item\n\nLEFT JOIN organizations o ON item.organization_id = o.id;"
  }, collection)

  // remove field
  collection.fields.removeById("_clone_rbBd")

  // add field
  collection.fields.addAt(8, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_D5zJ",
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
    "viewQuery": "SELECT\n  item.id,\n  item.type,\n  item.name,\n  item.description,\n  item.updated,\n  item.avatar,\n  item.avatar_url,\n  item.organization_id,\n  o.name AS organization_name\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    NULL AS avatar,\n    w.logo_url AS avatar_url,\n    'wallets' AS type,\n    w.owner AS organization_id\n  FROM wallets w\n  WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION ALL\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type,\n    ci.owner AS organization_id\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n\n  UNION ALL\n\n  SELECT\n    cr.id AS id,\n    COALESCE(NULLIF(cr.name, ''), cr.key) AS name,\n    NULL AS description,\n    cr.updated AS updated,\n    NULL AS avatar,\n    cr.logo AS avatar_url,\n    'credentials' AS type,\n    cr.owner AS organization_id\n  FROM credentials cr\n  JOIN credential_issuers AS i ON cr.credential_issuer = i.id\n  WHERE cr.name IS NOT NULL AND cr.published = true AND i.published = true\n\n  UNION ALL\n\n  SELECT\n    cc.id AS id,\n    cc.name AS name,\n    cc.description AS description,\n    cc.updated AS updated,\n    JSON_OBJECT(\n        'id', cc.id,\n        'collectionId', (SELECT id FROM _collections WHERE name = 'custom_checks'), \n        'collectionName', 'custom_checks', \n        'image_file', cc.logo\n    ) AS avatar,\n    NULL AS avatar_url,\n    'custom_checks' AS type, \n    cc.owner AS organization_id\n  FROM custom_checks cc\n  WHERE cc.name IS NOT NULL AND cc.public = true\n\n  UNION ALL\n\n  SELECT\n    vr.id AS id,\n    vr.name AS name,\n    vr.description AS description,\n    vr.updated AS updated,\n    JSON_OBJECT(\n      'id', vr.id,\n      'collectionId', (SELECT id FROM _collections WHERE name = 'verifiers'), \n      'collectionName', 'verifiers', \n      'image_file', vr.logo\n    ) AS avatar,\n    NULL AS avatar_url,\n    'verifiers' AS type, \n    vr.owner AS organization_id\n  FROM verifiers as vr\n  WHERE vr.name IS NOT NULL AND vr.published = true\n\n  UNION ALL\n\n  SELECT\n    vuc.id AS id,\n    vuc.name AS name,\n    vuc.description AS description,\n    vuc.updated as updated,\n    NULL as avatar,\n    NULL as avatar_url,\n    'use_cases_verifications' as type,\n    vuc.owner as organization_id\n  FROM use_cases_verifications as vuc\n  JOIN verifiers as v ON vuc.verifier = v.id\n  WHERE vuc.published = true AND v.published = true\n) AS item\n\nLEFT JOIN organizations o ON item.organization_id = o.id;"
  }, collection)

  // add field
  collection.fields.addAt(8, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_rbBd",
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
  collection.fields.removeById("_clone_D5zJ")

  return app.save(collection)
})
