/// <reference path="../pb_data/types.d.ts" />
// Runners no longer serve organization namespaces, so organizations need no
// published flag: public organizations are the ones listed in hub_organizations.
migrate((app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  collection.fields.removeById("bool1748787223")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("aako88kt3br4npt")

  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "bool1748787223",
    "name": "published",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  app.save(collection)

  app.db()
    .newQuery("UPDATE organizations SET published = (id IN (SELECT id FROM hub_organizations))")
    .execute()
})
