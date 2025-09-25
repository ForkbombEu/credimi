/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_183765882");

        unmarshal(
            {
                indexes: [
                    "CREATE UNIQUE INDEX `idx_fihXiaFPhk` ON `credentials` (`name`, `credential_issuer`)",
                    "CREATE UNIQUE INDEX `idx_woo8qWjgVH` ON `credentials` (`canonified_name`)",
                ],
            },
            collection,
        );

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_183765882");

        unmarshal(
            {
                indexes: [
                    "CREATE UNIQUE INDEX `idx_fihXiaFPhk` ON `credentials` (`key`, `credential_issuer`)",
                ],
            },
            collection,
        );

        return app.save(collection);
    },
);
