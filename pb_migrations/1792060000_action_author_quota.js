/// <reference path="../pb_data/types.d.ts" />
// Runners report their action author daily limits; Credimi counts usage per
// runner, pool (the runner's own organization or every other one) and UTC day.
// Tenants cannot write the limits through the API: only the runner lifecycle
// and registration handlers set them.
const agentServerFields = "@request.body.agent_own_daily_tokens:isset = false &&\n@request.body.agent_others_daily_tokens:isset = false &&\n@request.body.agent_provider:isset = false &&\n@request.body.agent_model:isset = false &&\n"

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
        "collectionId": "pbc_500646217",
        "hidden": false,
        "id": "relation1792060005",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "runner",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "relation"
      },
      {
        "hidden": false,
        "id": "select1792060006",
        "maxSelect": 1,
        "name": "pool",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "select",
        "values": [
          "own",
          "others"
        ]
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
      "CREATE UNIQUE INDEX `idx_action_author_usage_runner_pool_day` ON `action_author_usage` (`runner`, `pool`, `day`)"
    ],
    "listRule": null,
    "name": "action_author_usage",
    "system": false,
    "type": "base",
    "updateRule": null,
    "viewRule": null
  });

  app.save(collection);

  const runners = app.findCollectionByNameOrId("pbc_500646217");

  for (const [id, name] of [
    ["number1792060001", "agent_own_daily_tokens"],
    ["number1792060002", "agent_others_daily_tokens"],
  ]) {
    runners.fields.addAt(999, new Field({
      "hidden": false,
      "id": id,
      "max": null,
      "min": 0,
      "name": name,
      "onlyInt": true,
      "presentable": false,
      "required": false,
      "system": false,
      "type": "number"
    }));
  }
  for (const [id, name] of [
    ["text1792060003", "agent_provider"],
    ["text1792060004", "agent_model"],
  ]) {
    runners.fields.addAt(999, new Field({
      "autogeneratePattern": "",
      "hidden": false,
      "id": id,
      "max": 0,
      "min": 0,
      "name": name,
      "pattern": "",
      "presentable": false,
      "primaryKey": false,
      "required": false,
      "system": false,
      "type": "text"
    }));
  }
  runners.createRule = agentServerFields + runners.createRule;
  runners.updateRule = agentServerFields + runners.updateRule;

  return app.save(runners);
}, (app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");

  for (const id of ["number1792060001", "number1792060002", "text1792060003", "text1792060004"]) {
    runners.fields.removeById(id);
  }
  for (const rule of ["createRule", "updateRule"]) {
    if (runners[rule].startsWith(agentServerFields)) {
      runners[rule] = runners[rule].slice(agentServerFields.length);
    }
  }

  app.save(runners);

  const collection = app.findCollectionByNameOrId("pbc_3061840921");

  return app.delete(collection);
})
