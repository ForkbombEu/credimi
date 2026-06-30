/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_500646217")

  // add field
  collection.fields.addAt(11, new Field({
    "hidden": false,
    "id": "bool2654125802",
    "name": "online",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  // add field
  collection.fields.addAt(12, new Field({
    "hidden": false,
    "id": "date4259687325",
    "max": "",
    "min": "",
    "name": "last_heartbeat_at",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "date"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_500646217")

  // remove field
  collection.fields.removeById("bool2654125802")

  // remove field
  collection.fields.removeById("date4259687325")

  return app.save(collection)
})
