/// <reference path="../pb_data/types.d.ts" />
// Organization owners may publish their runners again: a runner's Temporal
// token now opens only its own namespace (runner-<record id>), so a published
// runner no longer serves every published organization's namespace.
const runnersId = "pbc_500646217";
const credentialGenerationGuard = "@request.body.credential_generation:isset = false &&\n";
const publishedGuard = "@request.body.published:isset = false &&\n";

migrate((app) => {
  const runners = app.findCollectionByNameOrId(runnersId);

  for (const rule of ["createRule", "updateRule"]) {
    runners[rule] = runners[rule].replace(publishedGuard, "");
  }

  return app.save(runners);
}, (app) => {
  const runners = app.findCollectionByNameOrId(runnersId);

  // 1792000000_restrict_mobile_runner_published.js placed the guard right
  // after the credential_generation guard.
  for (const rule of ["createRule", "updateRule"]) {
    runners[rule] = runners[rule].replace(
      credentialGenerationGuard,
      credentialGenerationGuard + publishedGuard,
    );
  }

  return app.save(runners);
});
