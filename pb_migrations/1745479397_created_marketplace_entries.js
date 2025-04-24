/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = new Collection({
    "createRule": null,
    "deleteRule": null,
    "fields": [
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text3208210256",
        "max": 0,
        "min": 0,
        "name": "id",
        "pattern": "^[a-z0-9]+$",
        "presentable": false,
        "primaryKey": true,
        "required": true,
        "system": true,
        "type": "text"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "_clone_7yKT",
        "max": 0,
        "min": 2,
        "name": "name",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "convertURLs": false,
        "hidden": false,
        "id": "_clone_B69a",
        "maxSize": 0,
        "name": "description",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "editor"
      },
      {
        "hidden": false,
        "id": "_clone_QMLj",
        "maxSelect": 1,
        "maxSize": 5242880,
        "mimeTypes": [
          "image/png",
          "image/jpeg",
          "image/webp",
          "image/svg+xml"
        ],
        "name": "avatar",
        "presentable": false,
        "protected": false,
        "required": false,
        "system": false,
        "thumbs": null,
        "type": "file"
      },
      {
        "hidden": false,
        "id": "json2190673250",
        "maxSize": 1,
        "name": "avatar_url",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "json"
      },
      {
        "hidden": false,
        "id": "json2363381545",
        "maxSize": 1,
        "name": "type",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "json"
      }
    ],
    "id": "pbc_2786561295",
    "indexes": [],
    "listRule": "",
    "name": "marketplace_entries",
    "system": false,
    "type": "view",
    "updateRule": null,
    "viewQuery": "SELECT\n  o.id AS id,\n  o.name AS name,\n  o.description AS description,\n  o.logo AS avatar,\n  NULL AS avatar_url,\n  'organization' AS type\nFROM organizations o\nWHERE o.name IS NOT NULL",
    "viewRule": ""
  });

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2786561295");

  return app.delete(collection);
})
