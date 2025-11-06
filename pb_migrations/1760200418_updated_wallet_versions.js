/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_2201295156");

        // update collection data
        unmarshal(
            {
                indexes: [
                    "CREATE UNIQUE INDEX `idx_jjqeX0KhkF` ON `wallet_versions` (\n  `wallet`,\n  `canonified_tag`,\n  `owner`\n)",
                ],
            },
            collection,
        );

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_2201295156");
        unmarshal(
            {
                indexes: [
                    "CREATE UNIQUE INDEX `idx_jjqeX0KhkF` ON `wallet_versions` (\n  `wallet`,\n  `tag`,\n  `owner`\n)",
                ],
            },
            collection,
        );

        return app.save(collection);
    },
);
