/// <reference path="../pb_data/types.d.ts" />
// Other organizations use published runners through Credimi's runner and
// device lists, which omit addresses and serials; the collection API shows
// runners and devices only to their organization.
const runnersId = "pbc_500646217";
const devicesId = "pbc_3594974968";
const orgRule =
  "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n" +
  "@collection.orgAuthorizations.organization.id ?= owner.id";
const previousOrgRule =
  "(\n" +
  "  @collection.orgAuthorizations.user.id ?= @request.auth.id &&\n" +
  "  @collection.orgAuthorizations.organization.id ?= owner.id\n" +
  ")";

migrate((app) => {
  const runners = app.findCollectionByNameOrId(runnersId);
  runners.listRule = orgRule;
  runners.viewRule = orgRule;
  app.save(runners);

  const devices = app.findCollectionByNameOrId(devicesId);
  devices.listRule = orgRule;
  devices.viewRule = orgRule;
  return app.save(devices);
}, (app) => {
  const runners = app.findCollectionByNameOrId(runnersId);
  runners.listRule = "published = true || " + previousOrgRule;
  runners.viewRule = "published = true || " + previousOrgRule;
  app.save(runners);

  const devices = app.findCollectionByNameOrId(devicesId);
  devices.listRule = "runner.published = true || " + previousOrgRule;
  devices.viewRule = "runner.published = true || " + previousOrgRule;
  return app.save(devices);
});
