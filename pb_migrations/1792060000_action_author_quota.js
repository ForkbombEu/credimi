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
        "cascadeDelete": true,
        "collectionId": "aako88kt3br4npt",
        "hidden": false,
        "id": "relation3253625724",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "organization",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "relation"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text2862495610",
        "max": 0,
        "min": 0,
        "name": "day",
        "pattern": "^\\d{4}-\\d{2}-\\d{2}$",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "hidden": false,
        "id": "number1907403416",
        "max": null,
        "min": 0,
        "name": "sessions",
        "onlyInt": true,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "number2381640387",
        "max": null,
        "min": 0,
        "name": "input_tokens",
        "onlyInt": true,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "number3524729851",
        "max": null,
        "min": 0,
        "name": "output_tokens",
        "onlyInt": true,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "number1290153472",
        "max": null,
        "min": 0,
        "name": "model_calls",
        "onlyInt": true,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "json1792070003",
        "maxSize": 65536,
        "name": "report_ids",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "json"
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
    "id": "pbc_3061840921",
    "indexes": [
      "CREATE UNIQUE INDEX `idx_action_author_usage_org_day` ON `action_author_usage` (`organization`, `day`)"
    ],
    "listRule": null,
    "name": "action_author_usage",
    "system": false,
    "type": "base",
    "updateRule": null,
    "viewRule": null
  });

  app.save(collection);

  const organizations = app.findCollectionByNameOrId("aako88kt3br4npt");

  // add field
  organizations.fields.addAt(999, new Field({
    "hidden": false,
    "id": "number1792070002",
    "max": null,
    "min": null,
    "name": "action_author_daily_tokens",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }));

  return app.save(organizations);
}, (app) => {
  const organizations = app.findCollectionByNameOrId("aako88kt3br4npt");

  // remove field
  organizations.fields.removeById("number1792070002");

  app.save(organizations);

  const collection = app.findCollectionByNameOrId("pbc_3061840921");

  return app.delete(collection);
})
