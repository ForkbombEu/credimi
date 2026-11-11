/// <reference path="../pb_data/types.d.ts" />
const ownerRole = "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\""
const runnerServerFields = "@request.body.admin_managed:isset = false &&\n@request.body.disabled:isset = false &&\n@request.body.online:isset = false &&\n@request.body.last_heartbeat_at:isset = false &&\n"

// credential_generation is bumped by the registration handler on every upsert
// and feeds the per-runner credential; tenant requests cannot set it.
const credentialGenerationGuard = "@request.body.credential_generation:isset = false &&\n"

// Only superusers publish runners: a published runner serves every published organization's namespace.
const publishedGuard = "@request.body.published:isset = false &&\n"

migrate((app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217")

  unmarshal({
    "createRule": runnerServerFields + credentialGenerationGuard + publishedGuard + ownerRole,
    "updateRule": runnerServerFields + credentialGenerationGuard + publishedGuard + "@request.body.owner:isset = false &&\n" + ownerRole
  }, runners)

  return app.save(runners)
}, (app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217")

  unmarshal({
    "createRule": runnerServerFields + credentialGenerationGuard + ownerRole,
    "updateRule": runnerServerFields + credentialGenerationGuard + "@request.body.owner:isset = false &&\n" + ownerRole
  }, runners)

  return app.save(runners)
})
