/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_183765882")

  // update field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "select3736761055",
    "maxSelect": 1,
    "name": "format",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "jwt_vc_json",
      "mso_mdoc",
      "vc+sd-jwt",
      "ldp_vc",
      "dc+sd-jwt"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_183765882")

  // update field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "select3736761055",
    "maxSelect": 1,
    "name": "format",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "jwt_vc_json",
      "mso_mdoc",
      "vc+sd-jwt",
      "ldp_vc"
    ]
  }))

  return app.save(collection)
})
