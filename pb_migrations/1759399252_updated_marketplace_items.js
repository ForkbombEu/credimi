/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT\n  item.id,\n  item.type,\n  item.name,\n  item.description,\n  item.updated,\n  item.avatar_file,\n  item.avatar_url,\n  item.organization_id,\n  item.children,\n  o.name AS organization_name\nFROM (\n  -- wallets\n  SELECT\n    w.id AS id,\n    'wallets' AS type,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    w.logo AS avatar_file,\n    w.logo_url AS avatar_url,\n    w.owner AS organization_id,\n    '[]' AS children\n  FROM wallets w\n  WHERE w.name IS NOT NULL AND w.published = 1\n\n  UNION ALL\n\n  -- credential_issuers (with child credentials)\n  SELECT\n    ci.id AS id,\n    'credential_issuers' AS type,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar_file,\n    ci.logo_url AS avatar_url,\n    ci.owner AS organization_id,\n    CASE \n      WHEN COUNT(cred.id) > 0 THEN \n        json_group_array(\n          json_object(\n            'id', cred.id,\n            'name', COALESCE(NULLIF(cred.display_name, ''), cred.name)\n          )\n        )\n      ELSE '[]'\n    END AS children\n  FROM credential_issuers ci\n  LEFT JOIN (\n    SELECT\n      id, display_name, name, credential_issuer\n    FROM credentials\n    WHERE published = 1\n    ORDER BY COALESCE(NULLIF(display_name, ''), name)\n  ) AS cred\n    ON cred.credential_issuer = ci.id\n  WHERE ci.name IS NOT NULL AND ci.published = 1\n  GROUP BY ci.id\n\n  UNION ALL\n\n  -- credentials\n  SELECT\n    cr.id AS id,\n    'credentials' AS type,\n    COALESCE(NULLIF(cr.display_name, ''), cr.name) AS name,\n    NULL AS description,\n    cr.updated AS updated,\n    NULL AS avatar_file,\n    cr.logo AS avatar_url,\n    cr.owner AS organization_id,\n    '[]' AS children\n  FROM credentials cr\n  JOIN credential_issuers AS i\n    ON cr.credential_issuer = i.id\n  WHERE cr.name IS NOT NULL AND cr.published = 1 AND i.published = 1\n\n  UNION ALL\n\n  -- custom_checks\n  SELECT\n    cc.id AS id,\n    'custom_checks' AS type,\n    cc.name AS name,\n    cc.description AS description,\n    cc.updated AS updated,\n    cc.logo AS avatar_file,\n    NULL AS avatar_url,\n    cc.owner AS organization_id,\n    '[]' AS children\n  FROM custom_checks cc\n  WHERE cc.name IS NOT NULL AND cc.public = 1\n\n  UNION ALL\n\n  -- verifiers (with child use_cases_verifications)\n  SELECT\n    vr.id AS id,\n    'verifiers' AS type,\n    vr.name AS name,\n    vr.description AS description,\n    vr.updated AS updated,\n    vr.logo AS avatar_file,\n    NULL AS avatar_url,\n    vr.owner AS organization_id,\n    CASE \n      WHEN COUNT(vuc.id) > 0 THEN \n        json_group_array(\n          json_object(\n            'id', vuc.id,\n            'name', vuc.name\n          )\n        )\n      ELSE '[]'\n    END AS children\n  FROM verifiers AS vr\n  LEFT JOIN (\n    SELECT id, name, verifier\n    FROM use_cases_verifications\n    WHERE published = 1\n    ORDER BY name\n  ) AS vuc\n    ON vuc.verifier = vr.id\n  WHERE vr.name IS NOT NULL AND vr.published = 1\n  GROUP BY vr.id\n\n  UNION ALL\n\n  -- use_cases_verifications\n  SELECT\n    vuc.id AS id,\n    'use_cases_verifications' AS type,\n    vuc.name AS name,\n    vuc.description AS description,\n    vuc.updated AS updated,\n    NULL AS avatar_file,\n    NULL AS avatar_url,\n    vuc.owner AS organization_id,\n    '[]' AS children\n  FROM use_cases_verifications AS vuc\n  JOIN verifiers AS v\n    ON vuc.verifier = v.id\n  WHERE vuc.published = 1 AND v.published = 1\n) AS item\nLEFT JOIN organizations o\n  ON item.organization_id = o.id;\n"
  }, collection)

  // remove field
  collection.fields.removeById("json376926767")

  // remove field
  collection.fields.removeById("_clone_NH6N")

  // add field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "json507207115",
    "maxSize": 1,
    "name": "avatar_file",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_MxhV",
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
    "viewQuery": "SELECT\n  item.id,\n  item.type,\n  item.name,\n  item.description,\n  item.updated,\n  item.avatar,\n  item.avatar_url,\n  item.organization_id,\n  item.children,\n  o.name AS organization_name\nFROM (\n  -- wallets\n  SELECT\n    w.id AS id,\n    'wallets' AS type,\n    w.name AS name,\n    w.description AS description,\n    w.updated AS updated,\n    w.logo AS avatar,\n    w.logo_url AS avatar_url,\n    w.owner AS organization_id,\n    '[]' AS children\n  FROM wallets w\n  WHERE w.name IS NOT NULL AND w.published = 1\n\n  UNION ALL\n\n  -- credential_issuers (with child credentials)\n  SELECT\n    ci.id AS id,\n    'credential_issuers' AS type,\n    ci.name AS name,\n    ci.description AS description,\n    ci.updated AS updated,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    ci.owner AS organization_id,\n    CASE \n      WHEN COUNT(cred.id) > 0 THEN \n        json_group_array(\n          json_object(\n            'id', cred.id,\n            'name', COALESCE(NULLIF(cred.display_name, ''), cred.name)\n          )\n        )\n      ELSE '[]'\n    END AS children\n  FROM credential_issuers ci\n  LEFT JOIN (\n    SELECT\n      id, display_name, name, credential_issuer\n    FROM credentials\n    WHERE published = 1\n    ORDER BY COALESCE(NULLIF(display_name, ''), name)\n  ) AS cred\n    ON cred.credential_issuer = ci.id\n  WHERE ci.name IS NOT NULL AND ci.published = 1\n  GROUP BY ci.id\n\n  UNION ALL\n\n  -- credentials\n  SELECT\n    cr.id AS id,\n    'credentials' AS type,\n    COALESCE(NULLIF(cr.display_name, ''), cr.name) AS name,\n    NULL AS description,\n    cr.updated AS updated,\n    NULL AS avatar,\n    cr.logo AS avatar_url,\n    cr.owner AS organization_id,\n    '[]' AS children\n  FROM credentials cr\n  JOIN credential_issuers AS i\n    ON cr.credential_issuer = i.id\n  WHERE cr.name IS NOT NULL AND cr.published = 1 AND i.published = 1\n\n  UNION ALL\n\n  -- custom_checks\n  SELECT\n    cc.id AS id,\n    'custom_checks' AS type,\n    cc.name AS name,\n    cc.description AS description,\n    cc.updated AS updated,\n    cc.logo AS avatar,\n    NULL AS avatar_url,\n    cc.owner AS organization_id,\n    '[]' AS children\n  FROM custom_checks cc\n  WHERE cc.name IS NOT NULL AND cc.public = 1\n\n  UNION ALL\n\n  -- verifiers (with child use_cases_verifications)\n  SELECT\n    vr.id AS id,\n    'verifiers' AS type,\n    vr.name AS name,\n    vr.description AS description,\n    vr.updated AS updated,\n    vr.logo AS avatar,\n    NULL AS avatar_url,\n    vr.owner AS organization_id,\n    CASE \n      WHEN COUNT(vuc.id) > 0 THEN \n        json_group_array(\n          json_object(\n            'id', vuc.id,\n            'name', vuc.name\n          )\n        )\n      ELSE '[]'\n    END AS children\n  FROM verifiers AS vr\n  LEFT JOIN (\n    SELECT id, name, verifier\n    FROM use_cases_verifications\n    WHERE published = 1\n    ORDER BY name\n  ) AS vuc\n    ON vuc.verifier = vr.id\n  WHERE vr.name IS NOT NULL AND vr.published = 1\n  GROUP BY vr.id\n\n  UNION ALL\n\n  -- use_cases_verifications\n  SELECT\n    vuc.id AS id,\n    'use_cases_verifications' AS type,\n    vuc.name AS name,\n    vuc.description AS description,\n    vuc.updated AS updated,\n    NULL AS avatar,\n    NULL AS avatar_url,\n    vuc.owner AS organization_id,\n    '[]' AS children\n  FROM use_cases_verifications AS vuc\n  JOIN verifiers AS v\n    ON vuc.verifier = v.id\n  WHERE vuc.published = 1 AND v.published = 1\n) AS item\nLEFT JOIN organizations o\n  ON item.organization_id = o.id;\n"
  }, collection)

  // add field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "json376926767",
    "maxSize": 1,
    "name": "avatar",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_NH6N",
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
  collection.fields.removeById("json507207115")

  // remove field
  collection.fields.removeById("_clone_MxhV")

  return app.save(collection)
})
