/// <reference path="../pb_data/types.d.ts" />
// Public organizations are the ones listed in hub_organizations: owners of a
// published wallet, credential issuer, verifier, custom check or pipeline.
// description and updated feed the sitemap and llms.txt.
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2405597765")

  unmarshal({
    "viewQuery": "SELECT id, name, canonified_name, logo, description, updated\nFROM organizations\nWHERE id IN (\n    SELECT owner FROM wallets WHERE published = TRUE\n    UNION\n    SELECT owner FROM credential_issuers WHERE published = TRUE\n    UNION\n    SELECT owner FROM verifiers WHERE published = TRUE\n    UNION\n    SELECT owner FROM custom_checks WHERE published = TRUE\n    UNION\n    SELECT owner FROM pipelines WHERE published = TRUE\n);"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2405597765")

  unmarshal({
    "viewQuery": "SELECT id, name, canonified_name, logo\nFROM organizations\nWHERE id IN (\n    SELECT owner FROM wallets WHERE published = TRUE\n    UNION\n    SELECT owner FROM credential_issuers WHERE published = TRUE\n    UNION\n    SELECT owner FROM verifiers WHERE published = TRUE\n    UNION\n    SELECT owner FROM custom_checks WHERE published = TRUE\n);"
  }, collection)

  return app.save(collection)
})
