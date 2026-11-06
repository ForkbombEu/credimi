/// <reference path="../pb_data/types.d.ts" />
// Credimi records who started each action author session: only that person
// writes in its chat, ends it or opens its live view.
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1792080000");

  collection.fields.addAt(999, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1792100001",
    "max": 0,
    "min": 0,
    "name": "actor_id",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }));

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1792080000");

  collection.fields.removeById("text1792100001");

  return app.save(collection);
})
