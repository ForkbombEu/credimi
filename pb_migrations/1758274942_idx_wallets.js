/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_120182150");

        unmarshal(
            {
                indexes: [
                    "CREATE UNIQUE INDEX `idx_nnTl8N8buH` ON `wallets` (`canonified_name`)",
                ],
            },
            collection,
        );

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_120182150");

        unmarshal(
            {
                indexes: [],
            },
            collection,
        );

        return app.save(collection);
    },
);
