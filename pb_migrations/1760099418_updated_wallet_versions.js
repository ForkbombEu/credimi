/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_2201295156");

        // add field
        collection.fields.addAt(
            4,
            new Field({
                autogeneratePattern: "",
                hidden: false,
                id: "text2621739057",
                max: 0,
                min: 0,
                name: "canonified_tag",
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
        const collection = app.findCollectionByNameOrId("pbc_2201295156");

        // remove field
        collection.fields.removeById("text2621739057");

        return app.save(collection);
    },
);
