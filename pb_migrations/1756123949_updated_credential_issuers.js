/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_678514665")

  // add field
  collection.fields.addAt(9, new Field({
    "exceptDomains": null,
    "hidden": false,
    "id": "url3023702915",
    "name": "workflow_url",
    "onlyDomains": null,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "url"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_678514665")

  // remove field
  collection.fields.removeById("url3023702915")

  return app.save(collection)
})
