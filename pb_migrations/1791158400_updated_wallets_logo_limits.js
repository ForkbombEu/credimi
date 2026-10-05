/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_120182150")

  // accept only bounded images, as the other logo fields do
  const logo = collection.fields.getByName("logo")
  logo.mimeTypes = [
    "image/png",
    "image/jpeg",
    "image/gif",
    "image/webp",
    "image/svg+xml"
  ]
  logo.maxSize = 2097152

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_120182150")

  const logo = collection.fields.getByName("logo")
  logo.mimeTypes = []
  logo.maxSize = 0

  return app.save(collection)
})
