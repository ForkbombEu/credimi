/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_678514665");

        // add field canonified_name (no index yet)
        collection.fields.addAt(
            5,
            new Field({
                autogeneratePattern: "",
                hidden: false,
                id: "text2077450625",
                max: 0,
                min: 0,
                name: "canonified_name",
                pattern: "",
                presentable: false,
                primaryKey: false,
                required: false,
                system: false,
                type: "text",
            }),
        );

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_678514665");

        // remove field on rollback
        collection.fields.removeById("text2077450625");

        return app.save(collection);
    },
);
