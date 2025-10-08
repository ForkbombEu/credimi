/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "WITH wallets_cte AS (\n  SELECT\n    w.id,\n    'wallets' AS type,\n    w.name,\n    w.description,\n    w.updated,\n    w.logo AS avatar_file,\n    w.logo_url AS avatar_url,\n    w.owner AS organization_id,\n    '[]' AS children,\n    w.canonified_name\n  FROM wallets w\n  WHERE w.name IS NOT NULL\n    AND w.published = 1\n),\ncredential_issuers_cte AS (\n  SELECT\n    ci.id,\n    'credential_issuers' AS type,\n    ci.name,\n    ci.description,\n    ci.updated,\n    NULL AS avatar_file,\n    ci.logo_url AS avatar_url,\n    ci.owner AS organization_id,\n    COALESCE(\n      json_group_array(\n        json_object(\n          'id', cred.id,\n          'name', COALESCE(NULLIF(cred.display_name, ''), cred.name),\n          'canonified_name', cred.canonified_name\n        )\n      ),\n      '[]'\n    ) AS children,\n    ci.canonified_name\n  FROM credential_issuers ci\n  LEFT JOIN credentials cred\n    ON cred.credential_issuer = ci.id\n   AND cred.published = 1\n  WHERE ci.name IS NOT NULL\n    AND ci.published = 1\n  GROUP BY ci.id\n),\ncredentials_cte AS (\n  SELECT\n    cr.id,\n    'credentials' AS type,\n    COALESCE(NULLIF(cr.display_name, ''), cr.name) AS name,\n    NULL AS description,\n    cr.updated,\n    NULL AS avatar_file,\n    cr.logo AS avatar_url,\n    cr.owner AS organization_id,\n    '[]' AS children,\n    cr.canonified_name\n  FROM credentials cr\n  JOIN credential_issuers i\n    ON cr.credential_issuer = i.id\n   AND i.published = 1\n  WHERE cr.name IS NOT NULL\n    AND cr.published = 1\n),\ncustom_checks_cte AS (\n  SELECT\n    cc.id,\n    'custom_checks' AS type,\n    cc.name,\n    cc.description,\n    cc.updated,\n    cc.logo AS avatar_file,\n    NULL AS avatar_url,\n    cc.owner AS organization_id,\n    '[]' AS children,\n    cc.canonified_name\n  FROM custom_checks cc\n  WHERE cc.name IS NOT NULL\n    AND cc.public = 1\n),\nverifiers_cte AS (\n  SELECT\n    vr.id,\n    'verifiers' AS type,\n    vr.name,\n    vr.description,\n    vr.updated,\n    vr.logo AS avatar_file,\n    NULL AS avatar_url,\n    vr.owner AS organization_id,\n    COALESCE(\n      json_group_array(\n        json_object(\n          'id', vuc.id,\n          'name', vuc.name,\n          'canonified_name', vuc.canonified_name\n        )\n      ),\n      '[]'\n    ) AS children,\n    vr.canonified_name\n  FROM verifiers vr\n  LEFT JOIN use_cases_verifications vuc\n    ON vuc.verifier = vr.id\n   AND vuc.published = 1\n  WHERE vr.name IS NOT NULL\n    AND vr.published = 1\n  GROUP BY vr.id\n),\nuse_cases_verifications_cte AS (\n  SELECT\n    vuc.id,\n    'use_cases_verifications' AS type,\n    vuc.name,\n    vuc.description,\n    vuc.updated,\n    NULL AS avatar_file,\n    NULL AS avatar_url,\n    vuc.owner AS organization_id,\n    '[]' AS children,\n    vuc.canonified_name\n  FROM use_cases_verifications vuc\n  JOIN verifiers v\n    ON vuc.verifier = v.id\n   AND v.published = 1\n  WHERE vuc.published = 1\n),\nunioned AS (\n  SELECT * FROM wallets_cte\n  UNION ALL\n  SELECT * FROM credential_issuers_cte\n  UNION ALL\n  SELECT * FROM credentials_cte\n  UNION ALL\n  SELECT * FROM custom_checks_cte\n  UNION ALL\n  SELECT * FROM verifiers_cte\n  UNION ALL\n  SELECT * FROM use_cases_verifications_cte\n)\nSELECT\n  u.id,\n  u.type,\n  u.name,\n  u.description,\n  u.updated,\n  u.avatar_file,\n  u.avatar_url,\n  u.organization_id,\n  u.children,\n  u.canonified_name,\n  o.name AS organization_name,\n  o.canonified_name AS organization_canonified_name,\n  CONCAT(o.canonified_name, \"/\", u.canonified_name) as path\nFROM unioned u\nLEFT JOIN organizations o\n  ON u.organization_id = o.id;\n"
  }, collection)

  // remove field
  collection.fields.removeById("_clone_nmmm")

  // add field
  collection.fields.addAt(10, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_5oWT",
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

  // add field
  collection.fields.addAt(11, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_nNz2",
    "max": 0,
    "min": 0,
    "name": "organization_canonified_name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(12, new Field({
    "hidden": false,
    "id": "json190089999",
    "maxSize": 1,
    "name": "path",
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
    "viewQuery": "WITH wallets_cte AS (\n  SELECT\n    w.id,\n    'wallets' AS type,\n    w.name,\n    w.description,\n    w.updated,\n    w.logo AS avatar_file,\n    w.logo_url AS avatar_url,\n    w.owner AS organization_id,\n    '[]' AS children,\n    w.canonified_name\n  FROM wallets w\n  WHERE w.name IS NOT NULL\n    AND w.published = 1\n),\ncredential_issuers_cte AS (\n  SELECT\n    ci.id,\n    'credential_issuers' AS type,\n    ci.name,\n    ci.description,\n    ci.updated,\n    NULL AS avatar_file,\n    ci.logo_url AS avatar_url,\n    ci.owner AS organization_id,\n    COALESCE(\n      json_group_array(\n        json_object(\n          'id', cred.id,\n          'name', COALESCE(NULLIF(cred.display_name, ''), cred.name),\n          'canonified_name', cred.canonified_name\n        )\n      ),\n      '[]'\n    ) AS children,\n    ci.canonified_name\n  FROM credential_issuers ci\n  LEFT JOIN credentials cred\n    ON cred.credential_issuer = ci.id\n   AND cred.published = 1\n  WHERE ci.name IS NOT NULL\n    AND ci.published = 1\n  GROUP BY ci.id\n),\ncredentials_cte AS (\n  SELECT\n    cr.id,\n    'credentials' AS type,\n    COALESCE(NULLIF(cr.display_name, ''), cr.name) AS name,\n    NULL AS description,\n    cr.updated,\n    NULL AS avatar_file,\n    cr.logo AS avatar_url,\n    cr.owner AS organization_id,\n    '[]' AS children,\n    cr.canonified_name\n  FROM credentials cr\n  JOIN credential_issuers i\n    ON cr.credential_issuer = i.id\n   AND i.published = 1\n  WHERE cr.name IS NOT NULL\n    AND cr.published = 1\n),\ncustom_checks_cte AS (\n  SELECT\n    cc.id,\n    'custom_checks' AS type,\n    cc.name,\n    cc.description,\n    cc.updated,\n    cc.logo AS avatar_file,\n    NULL AS avatar_url,\n    cc.owner AS organization_id,\n    '[]' AS children,\n    cc.canonified_name\n  FROM custom_checks cc\n  WHERE cc.name IS NOT NULL\n    AND cc.public = 1\n),\nverifiers_cte AS (\n  SELECT\n    vr.id,\n    'verifiers' AS type,\n    vr.name,\n    vr.description,\n    vr.updated,\n    vr.logo AS avatar_file,\n    NULL AS avatar_url,\n    vr.owner AS organization_id,\n    COALESCE(\n      json_group_array(\n        json_object(\n          'id', vuc.id,\n          'name', vuc.name,\n          'canonified_name', vuc.canonified_name\n        )\n      ),\n      '[]'\n    ) AS children,\n    vr.canonified_name\n  FROM verifiers vr\n  LEFT JOIN use_cases_verifications vuc\n    ON vuc.verifier = vr.id\n   AND vuc.published = 1\n  WHERE vr.name IS NOT NULL\n    AND vr.published = 1\n  GROUP BY vr.id\n),\nuse_cases_verifications_cte AS (\n  SELECT\n    vuc.id,\n    'use_cases_verifications' AS type,\n    vuc.name,\n    vuc.description,\n    vuc.updated,\n    NULL AS avatar_file,\n    NULL AS avatar_url,\n    vuc.owner AS organization_id,\n    '[]' AS children,\n    vuc.canonified_name\n  FROM use_cases_verifications vuc\n  JOIN verifiers v\n    ON vuc.verifier = v.id\n   AND v.published = 1\n  WHERE vuc.published = 1\n),\nunioned AS (\n  SELECT * FROM wallets_cte\n  UNION ALL\n  SELECT * FROM credential_issuers_cte\n  UNION ALL\n  SELECT * FROM credentials_cte\n  UNION ALL\n  SELECT * FROM custom_checks_cte\n  UNION ALL\n  SELECT * FROM verifiers_cte\n  UNION ALL\n  SELECT * FROM use_cases_verifications_cte\n)\nSELECT\n  u.id,\n  u.type,\n  u.name,\n  u.description,\n  u.updated,\n  u.avatar_file,\n  u.avatar_url,\n  u.organization_id,\n  u.children,\n  u.canonified_name,\n  o.name AS organization_name\nFROM unioned u\nLEFT JOIN organizations o\n  ON u.organization_id = o.id;\n"
  }, collection)

  // add field
  collection.fields.addAt(10, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_nmmm",
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
  collection.fields.removeById("_clone_5oWT")

  // remove field
  collection.fields.removeById("_clone_nNz2")

  // remove field
  collection.fields.removeById("json190089999")

  return app.save(collection)
})
