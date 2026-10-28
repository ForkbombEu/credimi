// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

migrate((app) => {
  const collection = app.findCollectionByNameOrId("wallet_actions");
  collection.fields.add(new JSONField({
    name: "authoring_provenance",
    hidden: true,
    maxSize: 65536,
  }));
  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("wallet_actions");
  collection.fields.removeByName("authoring_provenance");
  return app.save(collection);
});
