/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = new Collection({
    "createRule": null,
    "deleteRule": null,
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
        "cascadeDelete": false,
        "collectionId": "pbc_2153234234",
        "hidden": false,
        "id": "relation2113722841",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "pipeline",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "relation"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text746335418",
        "max": 0,
        "min": 0,
        "name": "workflow_id",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text2229534404",
        "max": 0,
        "min": 0,
        "name": "run_id",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "hidden": false,
        "id": "file2051935673",
        "maxSelect": 99,
        "maxSize": 0,
        "mimeTypes": [
          "video/mpeg",
          "video/mp4",
          "video/webm"
        ],
        "name": "video_results",
        "presentable": false,
        "protected": false,
        "required": false,
        "system": false,
        "thumbs": [],
        "type": "file"
      },
      {
        "hidden": false,
        "id": "file445458195",
        "maxSelect": 99,
        "maxSize": 0,
        "mimeTypes": [],
        "name": "screenshots",
        "presentable": false,
        "protected": false,
        "required": false,
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
    "id": "pbc_2980015441",
    "indexes": [
      "CREATE UNIQUE INDEX `idx_jfofwpkXdf` ON `pipeline_results` (\n  `owner`,\n  `pipeline`\n)",
      "CREATE UNIQUE INDEX `idx_QE0nzfgj8n` ON `pipeline_results` (\n  `owner`,\n  `workflow_id`,\n  `run_id`\n)"
    ],
    "listRule": null,
    "name": "pipeline_results",
    "system": false,
    "type": "base",
    "updateRule": null,
    "viewRule": null
  });

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2980015441");

  return app.delete(collection);
})
