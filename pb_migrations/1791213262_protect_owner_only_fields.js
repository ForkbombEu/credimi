/// <reference path="../pb_data/types.d.ts" />
// Owner-only fields must not be filterable, sortable or downloadable by
// non-owners. Hidden fields are shown to owners by the recordsecrets and
// wallet_versions enrich hooks, and their writes are loaded by the matching
// create/update request hooks. Installer downloads are gated by the
// wallet_versions file download hook.
//
// Hidden fields are dropped from @request.body, so the "at least one
// installer" check moves from the wallet_versions create rule to the
// create request hook.
const ownerRule = `@collection.orgAuthorizations.user.id ?= @request.auth.id &&
@collection.orgAuthorizations.organization.id ?= owner.id &&
@collection.orgAuthorizations.role.name ?= "owner"`

const installerCreateRule = `${ownerRule} &&
(
  @request.body.android_installer:isset = true ||
  @request.body.ios_installer:isset = true
)`

function setOwnerOnly(app, ownerOnly) {
  for (const name of ["credentials", "use_cases_verifications"]) {
    const collection = app.findCollectionByNameOrId(name)
    collection.fields.getByName("secrets").hidden = ownerOnly
    app.save(collection)
  }

  const versions = app.findCollectionByNameOrId("pbc_2201295156")
  for (const name of ["android_installer", "ios_installer"]) {
    const field = versions.fields.getByName(name)
    field.hidden = ownerOnly
    field.protected = ownerOnly
  }
  versions.createRule = ownerOnly ? ownerRule : installerCreateRule

  return app.save(versions)
}

migrate((app) => setOwnerOnly(app, true), (app) => setOwnerOnly(app, false))
