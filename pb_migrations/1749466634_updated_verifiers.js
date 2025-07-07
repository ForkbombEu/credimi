/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_131690875")

  // update field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "select384687787",
    "maxSelect": 7,
    "name": "signing_algorithms",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "ES256",
      "EdDSA",
      "Ed25519Signature2020",
      "RS256",
      "ES256K",
      "RSA",
      "RsaSignature2018"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_131690875")

  // update field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "select384687787",
    "maxSelect": 2,
    "name": "signing_algorithms",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "ES256",
      "EdDSA",
      "Ed25519Signature2020",
      "RS256",
      "ES256K",
      "RSA",
      "RsaSignature2018"
    ]
  }))

  return app.save(collection)
})
