/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = new Collection({
    "createRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\"",
    "deleteRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\"",
    "fields": [
      {
        "autogeneratePattern": "[a-z0-9]{15}",
        "hidden": false,
        "id": "text3208210256",
        "max": 15,
        "min": 15,
        "name": "id",
        "pattern": "^[a-z0-9]+$",
        "presentable": false,
        "primaryKey": true,
        "required": true,
        "system": true,
        "type": "text"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_120182150",
        "hidden": false,
        "id": "relation2087227935",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "wallet",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "relation"
      },
      {
        "cascadeDelete": false,
        "collectionId": "aako88kt3br4npt",
        "hidden": false,
        "id": "relation3479234172",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "owner",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "relation"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text59357059",
        "max": 0,
        "min": 0,
        "name": "tag",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "hidden": false,
        "id": "select961728715",
        "maxSelect": 1,
        "name": "platform",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "select",
        "values": [
          "android",
          "ios"
        ]
      },
      {
        "hidden": false,
        "id": "file2359244304",
        "maxSelect": 1,
        "maxSize": 262144000,
        "mimeTypes": [],
        "name": "file",
        "presentable": false,
        "protected": false,
        "required": true,
        "system": false,
        "thumbs": [],
        "type": "file"
      },
      {
        "hidden": false,
        "id": "autodate2990389176",
        "name": "created",
        "onCreate": true,
        "onUpdate": false,
        "presentable": false,
        "system": false,
        "type": "autodate"
      },
      {
        "hidden": false,
        "id": "autodate3332085495",
        "name": "updated",
        "onCreate": true,
        "onUpdate": true,
        "presentable": false,
        "system": false,
        "type": "autodate"
      }
    ],
    "id": "pbc_2201295156",
    "indexes": [
      "CREATE UNIQUE INDEX `idx_jjqeX0KhkF` ON `wallet_versions` (\n  `wallet`,\n  `tag`,\n  `owner`\n)"
    ],
    "listRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id",
    "name": "wallet_versions",
    "system": false,
    "type": "base",
    "updateRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id &&\n@collection.orgAuthorizations.role.name ?= \"owner\"",
    "viewRule": "@collection.orgAuthorizations.user.id ?= @request.auth.id &&\n@collection.orgAuthorizations.organization.id ?= owner.id"
  });

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2201295156");

  return app.delete(collection);
})
