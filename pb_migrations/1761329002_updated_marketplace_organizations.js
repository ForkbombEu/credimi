/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2405597765")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT id, name, canonified_name\nFROM organizations\nWHERE id IN (\n    SELECT owner FROM wallets WHERE published = TRUE\n    UNION\n    SELECT owner FROM credential_issuers WHERE published = TRUE\n    UNION\n    SELECT owner FROM credentials WHERE published = TRUE\n    UNION\n    SELECT owner FROM verifiers WHERE published = TRUE\n    UNION\n    SELECT owner FROM use_cases_verifications WHERE published = TRUE\n    UNION\n     SELECT owner FROM custom_checks WHERE published = TRUE\n);"
  }, collection)

  // add field
  collection.fields.addAt(1, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_Z3Lv",
    "max": 0,
    "min": 2,
    "name": "name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_KsOp",
    "max": 0,
    "min": 0,
    "name": "canonified_name",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2405597765")

  // update collection data
  unmarshal({
    "viewQuery": "SELECT id FROM organizations"
  }, collection)

  // remove field
  collection.fields.removeById("_clone_Z3Lv")

  // remove field
  collection.fields.removeById("_clone_KsOp")

  return app.save(collection)
})
