/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_131690875");

        unmarshal(
            {
                indexes: [
                    "CREATE UNIQUE INDEX `idx_3mgbTTqvSj` ON `verifiers` (`canonified_name`)",
                ],
            },
            collection,
        );

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_131690875");

        unmarshal(
            {
                indexes: [],
            },
            collection,
        );

        return app.save(collection);
    },
);
