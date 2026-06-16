/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153234234")

  const existingField = collection.fields.find((f) => f.name === 'manual')
  if (!existingField) {
    // add field
    collection.fields.addAt(8, new Field({
      "hidden": false,
      "id": "bool1781613904",
      "name": "manual",
      "presentable": false,
      "required": false,
      "system": false,
      "type": "bool"
    }))
  }

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153234234")

  const existingField = collection.fields.find((f) => f.name === 'manual')
  if (existingField) {
    // remove field
    collection.fields.removeById(existingField.id)
  }

  return app.save(collection)
})

