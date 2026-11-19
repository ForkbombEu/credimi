/// <reference path="../pb_data/types.d.ts" />
// Every runner that is not disabled owns the Temporal namespace
// runner-<record id>, whatever its agent: the hidden readiness flag is the
// runner namespace's, not the agent's.
const runnersId = "pbc_500646217";
const fieldId = "bool1792140001";

const rename = (app, from, to) => {
  const runners = app.findCollectionByNameOrId(runnersId);

  runners.fields.getById(fieldId).name = to;
  for (const rule of ["createRule", "updateRule"]) {
    runners[rule] = runners[rule].replace(
      `@request.body.${from}:isset = false`,
      `@request.body.${to}:isset = false`,
    );
  }

  return app.save(runners);
};

migrate(
  (app) => rename(app, "agent_namespace_ready", "runner_namespace_ready"),
  (app) => rename(app, "runner_namespace_ready", "agent_namespace_ready"),
);
