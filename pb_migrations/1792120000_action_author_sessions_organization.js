/// <reference path="../pb_data/types.d.ts" />
// Action author sessions run in their runner's namespace (runner-<record id>);
// Credimi records the organization (canonified_name) that started each one.
// Every earlier row ran in its organization's namespace. Session workflow IDs
// (ActionAuthorSession-<uuid>) are unique across namespaces.
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1792080000");

  collection.fields.addAt(999, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1792120001",
    "max": 0,
    "min": 0,
    "name": "organization",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }));
  collection.indexes = [
    "CREATE UNIQUE INDEX `idx_action_author_sessions_workflow` ON `action_author_sessions` (`workflow_id`)"
  ];
  app.save(collection);

  app.db()
    .newQuery("UPDATE action_author_sessions SET organization = namespace WHERE organization = ''")
    .execute();
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1792080000");

  collection.indexes = [
    "CREATE UNIQUE INDEX `idx_action_author_sessions_namespace_workflow` ON `action_author_sessions` (`namespace`, `workflow_id`)"
  ];
  collection.fields.removeById("text1792120001");

  return app.save(collection);
})
