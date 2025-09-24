/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("aako88kt3br4npt");

        unmarshal(
            {
                indexes: [
                    "CREATE UNIQUE INDEX `idx_kYKHruDMrW` ON `organizations` (`canonified_name`)",
                ],
            },
            collection,
        );

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("aako88kt3br4npt");

        unmarshal(
            {
                indexes: [],
            },
            collection,
        );

        return app.save(collection);
    },
);
