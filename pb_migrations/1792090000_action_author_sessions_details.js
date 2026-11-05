/// <reference path="../pb_data/types.d.ts" />
// Credimi lists a wallet's recorded action author sessions with the build and
// the device each one ran on.
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1792080000");

  collection.fields.addAt(999, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1792090001",
    "max": 0,
    "min": 0,
    "name": "wallet_version_identifier",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }));
  collection.fields.addAt(999, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1792090002",
    "max": 0,
    "min": 0,
    "name": "device_id",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }));

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1792080000");

  collection.fields.removeById("text1792090001");
  collection.fields.removeById("text1792090002");

  return app.save(collection);
})
