/// <reference path="../pb_data/types.d.ts" />
// Runners report whether their action author agent is enabled; Credimi
// prepares the runner's agent namespace only then. Tenants cannot write it
// through the API: only the runner lifecycle and registration handlers set it.
//
// The up migration also restores the agent guards of 1792060000 and
// 1792070000 on databases where 1792000000 ran after them and rewrote the
// rules without those guards.
const agentGuardFields = [
  "agent_own_daily_tokens",
  "agent_others_daily_tokens",
  "agent_provider",
  "agent_model",
  "agent_settings_from",
  "agent_enabled",
]
const agentEnabledGuard = "@request.body.agent_enabled:isset = false &&\n"

migrate((app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");

  runners.fields.addAt(999, new Field({
    "hidden": false,
    "id": "bool1792110001",
    "name": "agent_enabled",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }));
  for (const rule of ["createRule", "updateRule"]) {
    for (const field of agentGuardFields) {
      if (!runners[rule].includes("@request.body." + field + ":isset")) {
        runners[rule] = "@request.body." + field + ":isset = false &&\n" + runners[rule];
      }
    }
  }

  return app.save(runners);
}, (app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");

  runners.fields.removeById("bool1792110001");
  for (const rule of ["createRule", "updateRule"]) {
    if (runners[rule].startsWith(agentEnabledGuard)) {
      runners[rule] = runners[rule].slice(agentEnabledGuard.length);
    }
  }

  return app.save(runners);
})
