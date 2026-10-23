/// <reference path="../pb_data/types.d.ts" />
// A wallet version is readable by members of its owner organization, and by
// anyone when its wallet is published and the version belongs to the wallet's
// own organization. Versions other organizations attach to a published wallet
// (for example temporary CI wallet APK versions) stay private to them.
//
// PocketBase checks protected file downloads against the view rule with only
// the ?token= file token as auth, which would lock out the Authorization header
// and Credimi-Api-Key callers HandleInstallerDownload admits (the internal
// admin key used by admin-managed runners). In the protectedFile context the
// rule passes and HandleInstallerDownload stays the only installer gate.
const readRule = `@request.context = "protectedFile" ||
(wallet.published = true && owner = wallet.owner) || (
  @collection.orgAuthorizations.user.id ?= @request.auth.id &&
  @collection.orgAuthorizations.organization.id ?= owner.id
)`

function setReadRules(app, rule) {
  const collection = app.findCollectionByNameOrId("pbc_2201295156")

  unmarshal({
    "listRule": rule,
    "viewRule": rule
  }, collection)

  return app.save(collection)
}

migrate((app) => setReadRules(app, readRule), (app) => setReadRules(app, ""))
