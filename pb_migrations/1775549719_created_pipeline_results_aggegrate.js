/// <reference path="../pb_data/types.d.ts" />
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
        "cascadeDelete": false,
        "collectionId": "pbc_2153234234",
        "hidden": false,
        "id": "relation2113722841",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "pipeline",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "relation"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_500646217",
        "hidden": false,
        "id": "relation977024222",
        "maxSelect": 999,
        "minSelect": 0,
        "name": "mobile_runners",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
      },
      {
        "hidden": false,
        "id": "number4263700706",
        "max": null,
        "min": null,
        "name": "total_runs",
        "onlyInt": false,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "number3961044466",
        "max": null,
        "min": null,
        "name": "total_successes",
        "onlyInt": false,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "number218026181",
        "max": null,
        "min": null,
        "name": "manually_executed_runs",
        "onlyInt": false,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "number1341865014",
        "max": null,
        "min": null,
        "name": "scheduled_runs",
        "onlyInt": false,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text909375167",
        "max": 0,
        "min": 0,
        "name": "minimum_running_time",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": false,
        "system": false,
        "type": "text"
      },
      {
        "hidden": false,
        "id": "date4272768884",
        "max": "",
        "min": "",
        "name": "first_execution",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "date"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_2980015441",
        "hidden": false,
        "id": "relation3634082342",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "latest_execution",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_2201295156",
        "hidden": false,
        "id": "relation3984192678",
        "maxSelect": 999,
        "minSelect": 0,
        "name": "wallet_versions",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_1403167086",
        "hidden": false,
        "id": "relation1950995705",
        "maxSelect": 999,
        "minSelect": 0,
        "name": "wallet_actions",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_183765882",
        "hidden": false,
        "id": "relation4194641934",
        "maxSelect": 999,
        "minSelect": 0,
        "name": "credentials",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_92944219",
        "hidden": false,
        "id": "relation3464138750",
        "maxSelect": 999,
        "minSelect": 0,
        "name": "use_case_verifications",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
      },
      {
        "hidden": false,
        "id": "json1214815598",
        "maxSize": 0,
        "name": "conformance_checks",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "json"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_1108732172",
        "hidden": false,
        "id": "relation128547761",
        "maxSelect": 999,
        "minSelect": 0,
        "name": "custom_integrations",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
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
      },
      {
        "hidden": false,
        "id": "autodate3332085495",
        "name": "updated",
        "onCreate": true,
        "onUpdate": true,
        "presentable": false,
        "system": false,
        "type": "autodate"
      }
    ],
    "id": "pbc_1919502272",
    "indexes": [
      "CREATE UNIQUE INDEX `idx_xD8UMaxUgN` ON `pipeline_results_aggegrate` (`pipeline`)"
    ],
    "listRule": null,
    "name": "pipeline_results_aggegrate",
    "system": false,
    "type": "base",
    "updateRule": null,
    "viewRule": null
  });

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1919502272");

  return app.delete(collection);
})
