/// <reference path="../pb_data/types.d.ts" />
// Stub only: field schema and path unique index are owned by
// pkg/conformancecatalog.EnsureCollection (catalogFields).
migrate((app) => {
  try {
    app.findCollectionByNameOrId("conformance_checks");
    return;
  } catch (_) {}

  const collection = new Collection({
    "id": "pbc_conformance_checks_catalog",
    "name": "conformance_checks",
    "type": "base",
    "system": false,
    "listRule": "",
    "viewRule": "",
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "fields": [
      {
        "autogeneratePattern": "[a-z0-9]{15}",
        "hidden": false,
        "id": "text3208210256",
        "max": 15,
        "min": 15,
        "name": "id",
        "pattern": "^[a-z0-9]+$",
        "presentable": false,
        "primaryKey": true,
        "required": true,
        "system": true,
        "type": "text"
      }
    ],
    "indexes": []
  });

  return app.save(collection);
}, (app) => {
  try {
    const collection = app.findCollectionByNameOrId("pbc_conformance_checks_catalog");
    return app.delete(collection);
  } catch (_) {
    return;
  }
})
