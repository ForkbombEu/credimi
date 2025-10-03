/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_678514665");

        unmarshal(
            {
                indexes: [
                    "CREATE UNIQUE INDEX `idx_hRrjNsbb0u` ON `credential_issuers` (`canonified_name`)",
                ],
            },
            collection,
        );

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_678514665");

        unmarshal(
            {
                indexes: [],
            },
            collection,
        );

        return app.save(collection);
    },
);
