/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_2201295156");
        // update collection data
        app.db()
            .newQuery(
                "UPDATE wallet_versions SET canonified_tag = lower(hex(randomblob(7)))",
            )
            .execute();

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_2201295156");
        return app.save(collection);
    },
);
