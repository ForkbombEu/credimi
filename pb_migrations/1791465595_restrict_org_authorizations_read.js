/// <reference path="../pb_data/types.d.ts" />
// Organization membership is readable only by the member themself or by
// members of the same organization.
//
// orgAuthorizations is a system collection: PocketBase rejects API rule
// changes on it (and changes of the system flag) in validated saves, so the
// rules are saved without validation. TestOrgAuthorizationsReadRulesScopeToMembers
// exercises the rules through the records API.
const memberRule = `user.id = @request.auth.id || (
  @collection.orgAuthorizations.user.id ?= @request.auth.id &&
  @collection.orgAuthorizations.organization.id ?= organization.id
)`

function setReadRules(app, rule) {
  const collection = app.findCollectionByNameOrId("k1vlx34o1x8tzno")

  unmarshal({
    "listRule": rule,
    "viewRule": rule
  }, collection)

  return app.saveNoValidate(collection)
}

migrate((app) => setReadRules(app, memberRule), (app) => setReadRules(app, ""))
