/// <reference path="../pb_data/types.d.ts" />
// The action author settings of a mobile runner: whether its agent is enabled,
// its model, its daily token limits, who may set them, whether Credimi
// prepared its agent namespace, and the usage the runner reports for today.
// Only Credimi writes them (the runner lifecycle and registration handlers):
// tenants cannot set them, nor a runner's record id, which names its agent
// namespace (`runner-<record id>`), through the API.
//
// Development databases of the action author branch already hold most of
// these fields, so every step is idempotent.
const runnersId = "pbc_500646217";

const textField = (id, name, hidden) => ({
  "autogeneratePattern": "",
  "hidden": hidden,
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
});

const numberField = (id, name, hidden) => ({
  "hidden": hidden,
  "id": id,
  "max": null,
  "min": 0,
  "name": name,
  "onlyInt": true,
  "presentable": false,
  "required": false,
  "system": false,
  "type": "number"
});

const boolField = (id, name, hidden) => ({
  "hidden": hidden,
  "id": id,
  "name": name,
  "presentable": false,
  "required": false,
  "system": false,
  "type": "bool"
});

const fields = [
  boolField("bool1792110001", "agent_enabled", false),
  textField("text1792060003", "agent_provider", false),
  textField("text1792060004", "agent_model", false),
  numberField("number1792060001", "agent_own_daily_tokens", false),
  numberField("number1792060002", "agent_others_daily_tokens", false),
  textField("text1792070001", "agent_settings_from", true),
  boolField("bool1792140001", "agent_namespace_ready", true),
  textField("text1792060010", "agent_usage_day", true),
  numberField("number1792060011", "agent_own_tokens_used", true),
  numberField("number1792060012", "agent_others_tokens_used", true),
];

const guard = (name) => `@request.body.${name}:isset = false &&\n`;

const rules = {
  "createRule": ["id", ...fields.map((field) => field.name)],
  "updateRule": fields.map((field) => field.name),
};

migrate((app) => {
  const runners = app.findCollectionByNameOrId(runnersId);

  for (const field of fields) {
    const existing = runners.fields.getByName(field.name);
    if (existing) {
      existing.hidden = field.hidden;
    } else {
      runners.fields.addAt(999, new Field(field));
    }
  }
  for (const [rule, names] of Object.entries(rules)) {
    for (const name of names) {
      if (!runners[rule].includes(`@request.body.${name}:isset = false`)) {
        runners[rule] = guard(name) + runners[rule];
      }
    }
  }
  app.save(runners);

  // Only development databases of the action author branch have these
  // collections: sessions are read from Temporal and the runner meters usage.
  for (const name of ["action_author_usage", "action_author_sessions"]) {
    try {
      app.delete(app.findCollectionByNameOrId(name));
    } catch (_) {}
  }
}, (app) => {
  const runners = app.findCollectionByNameOrId(runnersId);

  for (const field of fields) {
    runners.fields.removeByName(field.name);
  }
  for (const [rule, names] of Object.entries(rules)) {
    for (const name of names) {
      runners[rule] = runners[rule].split(guard(name)).join("");
    }
  }

  return app.save(runners);
})
