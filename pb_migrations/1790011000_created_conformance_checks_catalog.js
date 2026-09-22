/// <reference path="../pb_data/types.d.ts" />
// Catalog listing is served from a process-private :memory: store via Credimi
// routes that mimic /api/collections/conformance_checks/records. Do not create
// a durable PocketBase collection for it.
migrate((app) => {
  const names = ["conformance_checks", "pbc_conformance_checks_catalog"];
  for (const name of names) {
    try {
      const collection = app.findCollectionByNameOrId(name);
      app.delete(collection);
    } catch (_) {}
  }
}, (app) => {
  // Irreversible on purpose: the durable catalog shell must not be restored.
})
