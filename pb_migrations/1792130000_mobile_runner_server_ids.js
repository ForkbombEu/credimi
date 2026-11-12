/// <reference path="../pb_data/types.d.ts" />
// A runner's agent namespace is `runner-<record id>`, so tenants cannot
// choose a runner record id through the API: a deleted runner's namespace
// must not be claimable by another organization. Credimi generates the id.
const serverIdGuard = "@request.body.id:isset = false &&\n"

migrate((app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");

  if (!runners.createRule.includes("@request.body.id:isset")) {
    runners.createRule = serverIdGuard + runners.createRule;
  }

  return app.save(runners);
}, (app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217");

  if (runners.createRule.startsWith(serverIdGuard)) {
    runners.createRule = runners.createRule.slice(serverIdGuard.length);
  }

  return app.save(runners);
})
