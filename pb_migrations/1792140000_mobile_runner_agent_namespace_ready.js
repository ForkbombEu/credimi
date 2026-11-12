/// <reference path="../pb_data/types.d.ts" />
// Credimi records on the runner that it prepared the runner's agent namespace,
// so a restarted Credimi answers heartbeats with the namespace before it has
// prepared it again. Only Credimi writes it: it is hidden and tenants cannot
// set it through the API.
const readyGuard = "@request.body.agent_namespace_ready:isset = false &&\n"

migrate((app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");

  runners.fields.addAt(999, new Field({
    "hidden": true,
    "id": "bool1792140001",
    "name": "agent_namespace_ready",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }));
  for (const rule of ["createRule", "updateRule"]) {
    if (!runners[rule].includes("@request.body.agent_namespace_ready:isset")) {
      runners[rule] = readyGuard + runners[rule];
    }
  }

  return app.save(runners);
}, (app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");

  runners.fields.removeById("bool1792140001");
  for (const rule of ["createRule", "updateRule"]) {
    if (runners[rule].startsWith(readyGuard)) {
      runners[rule] = runners[rule].slice(readyGuard.length);
    }
  }

  return app.save(runners);
})
