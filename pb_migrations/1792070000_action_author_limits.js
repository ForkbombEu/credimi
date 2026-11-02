/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const organizations = app.findCollectionByNameOrId("aako88kt3br4npt")

  // add field
  organizations.fields.addAt(999, new Field({
    "hidden": false,
    "id": "number1792070001",
    "max": null,
    "min": null,
    "name": "action_author_daily_sessions",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  organizations.fields.addAt(999, new Field({
    "hidden": false,
    "id": "number1792070002",
    "max": null,
    "min": null,
    "name": "action_author_daily_tokens",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  app.save(organizations)

  // No organization has an allowance until a superuser sets one.
  app.db().newQuery(
    "UPDATE organizations SET action_author_daily_sessions = 0, action_author_daily_tokens = 0"
  ).execute()

  const usage = app.findCollectionByNameOrId("pbc_3061840921")

  // add field
  usage.fields.addAt(999, new Field({
    "hidden": false,
    "id": "json1792070003",
    "maxSize": 65536,
    "name": "report_ids",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "json"
  }))

  return app.save(usage)
}, (app) => {
  const usage = app.findCollectionByNameOrId("pbc_3061840921")

  // remove field
  usage.fields.removeById("json1792070003")

  app.save(usage)

  const organizations = app.findCollectionByNameOrId("aako88kt3br4npt")

  // remove field
  organizations.fields.removeById("number1792070001")
  organizations.fields.removeById("number1792070002")

  return app.save(organizations)
})
