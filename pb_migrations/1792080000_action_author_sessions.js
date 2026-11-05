/// <reference path="../pb_data/types.d.ts" />
// Credimi records the one run of each action author session workflow it
// starts; activities accept only that run. No API rules: only Credimi reads
// and writes the collection. agent_settings_from is server-only too, so it
// is hidden from API responses.
migrate((app) => {
  const collection = new Collection({
    "createRule": null,
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
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text1792080001",
        "max": 0,
        "min": 0,
        "name": "namespace",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text1792080002",
        "max": 0,
        "min": 0,
        "name": "workflow_id",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text1792080003",
        "max": 0,
        "min": 0,
        "name": "run_id",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "hidden": false,
        "id": "autodate2990389176",
        "name": "created",
        "onCreate": true,
        "onUpdate": false,
        "presentable": false,
        "system": false,
        "type": "autodate"
      }
    ],
    "id": "pbc_1792080000",
    "indexes": [
      "CREATE UNIQUE INDEX `idx_action_author_sessions_namespace_workflow` ON `action_author_sessions` (`namespace`, `workflow_id`)"
    ],
    "listRule": null,
    "name": "action_author_sessions",
    "system": false,
    "type": "base",
    "updateRule": null,
    "viewRule": null
  });

  app.save(collection);

  const runners = app.findCollectionByNameOrId("pbc_500646217");
  runners.fields.getById("text1792070001").hidden = true;

  return app.save(runners);
}, (app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");
  runners.fields.getById("text1792070001").hidden = false;

  app.save(runners);

  const collection = app.findCollectionByNameOrId("pbc_1792080000");

  return app.delete(collection);
})
