/// <reference path="../pb_data/types.d.ts" />
const ownerRole = "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\""
const devicesOwnerRole = "  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n  @collection.orgAuthorizations.organization.id ?= owner.id &&\n  @collection.orgAuthorizations.role.name ?= \"owner\""

// Tenant requests cannot touch fields owned by superusers or server code:
// admin_managed and disabled (superuser), online and last_heartbeat_at
// (runner lifecycle handlers), and the owner/runner binding (registration
// handlers). Superusers and server-side saves bypass these rules.
const runnerServerFields = "@request.body.admin_managed:isset = false &&\n@request.body.disabled:isset = false &&\n@request.body.online:isset = false &&\n@request.body.last_heartbeat_at:isset = false &&\n"

migrate((app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217")
  unmarshal({
    "createRule": runnerServerFields + ownerRole,
    "updateRule": runnerServerFields + "@request.body.owner:isset = false &&\n" + ownerRole
  }, runners)
  app.save(runners)

  const devices = app.findCollectionByNameOrId("pbc_3594974968")
  unmarshal({
    "createRule": "@request.body.online:isset = false &&\n@request.body.runner.owner = @request.body.owner &&\n" + devicesOwnerRole,
    "updateRule": "@request.body.online:isset = false &&\n@request.body.owner:isset = false &&\n@request.body.runner:isset = false &&\n" + devicesOwnerRole
  }, devices)
  return app.save(devices)
}, (app) => {
  const runners = app.findCollectionByNameOrId("pbc_500646217")
  unmarshal({
    "createRule": ownerRole,
    "updateRule": ownerRole
  }, runners)
  app.save(runners)

  const devices = app.findCollectionByNameOrId("pbc_3594974968")
  unmarshal({
    "createRule": devicesOwnerRole,
    "updateRule": devicesOwnerRole
  }, devices)
  return app.save(devices)
})
