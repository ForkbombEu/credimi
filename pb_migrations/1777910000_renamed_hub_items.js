/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  // Rename marketplace_items -> hub_items
  const hubItems = app.findCollectionByNameOrId("pbc_2786561295");
  unmarshal({ "name": "hub_items" }, hubItems);
  app.save(hubItems);

  // Rename marketplace_organizations -> hub_organizations
  const hubOrgs = app.findCollectionByNameOrId("pbc_2405597765");
  unmarshal({ "name": "hub_organizations" }, hubOrgs);
  app.save(hubOrgs);
}, (app) => {
  // Rename hub_items -> marketplace_items
  const hubItems = app.findCollectionByNameOrId("pbc_2786561295");
  unmarshal({ "name": "marketplace_items" }, hubItems);
  app.save(hubItems);

  // Rename hub_organizations -> marketplace_organizations
  const hubOrgs = app.findCollectionByNameOrId("pbc_2405597765");
  unmarshal({ "name": "marketplace_organizations" }, hubOrgs);
  app.save(hubOrgs);
})
