/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_1108732172");

        unmarshal(
            {
                indexes: [
                    "CREATE UNIQUE INDEX `idx_kf6cgyLMvY` ON `custom_checks` (`canonified_name`)",
                ],
            },
            collection,
        );

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_1108732172");

        unmarshal(
            {
                indexes: [],
            },
            collection,
        );

        return app.save(collection);
    },
);
