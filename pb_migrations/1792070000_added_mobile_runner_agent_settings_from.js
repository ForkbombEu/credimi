/// <reference path="../pb_data/types.d.ts" />
// Credimi accepts a runner's action author settings only from the identity
// that registered it ("<auth collection>:<record id>"). Tenants cannot write
// that identity through the API: only the registration handler sets it.
const settingsFromGuard = "@request.body.agent_settings_from:isset = false &&\n"

migrate((app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");

  runners.fields.addAt(999, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1792070001",
    "max": 0,
    "min": 0,
    "name": "agent_settings_from",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }));
  runners.createRule = settingsFromGuard + runners.createRule;
  runners.updateRule = settingsFromGuard + runners.updateRule;

  return app.save(runners);
}, (app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");

  runners.fields.removeById("text1792070001");
  for (const rule of ["createRule", "updateRule"]) {
    if (runners[rule].startsWith(settingsFromGuard)) {
      runners[rule] = runners[rule].slice(settingsFromGuard.length);
    }
  }

  return app.save(runners);
})
