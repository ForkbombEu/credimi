/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT id, type, name, description, avatar, avatar_url\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.logo AS avatar,\n    NULL AS avatar_url,\n    'wallets' AS type\n  FROM wallets w WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n)"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT id, name, description, avatar, avatar_url, type\nFROM (\n  SELECT\n    w.id AS id,\n    w.name AS name,\n    w.description AS description,\n    w.logo AS avatar,\n    NULL AS avatar_url,\n    'wallets' AS type\n  FROM wallets w WHERE w.name IS NOT NULL AND w.published = true\n\n  UNION\n\n  SELECT\n    ci.id AS id,\n    ci.name AS name,\n    ci.description AS description,\n    NULL AS avatar,\n    ci.logo_url AS avatar_url,\n    'credential_issuers' AS type\n  FROM credential_issuers ci\n  WHERE ci.name IS NOT NULL AND ci.published = true\n)"
  }, collection)

  return app.save(collection)
})
