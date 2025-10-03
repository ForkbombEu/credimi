/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_183765882");

        // ✨ explicitly clear indexes so PB doesn’t try to recreate the old ones
        unmarshal(
            {
                indexes: [],
            },
            collection,
        );

        // add canonified_name
        collection.fields.addAt(
            13,
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

        // add display_name
        collection.fields.addAt(
            5,
            new Field({
                autogeneratePattern: "",
                hidden: false,
                id: "text1579384326",
                max: 0,
                min: 0,
                name: "display_name",
                pattern: "",
                presentable: false,
                primaryKey: false,
                required: false,
                system: false,
                type: "text",
            }),
        );

        // rename key → name
        collection.fields.addAt(
            12,
            new Field({
                autogeneratePattern: "",
                hidden: false,
                id: "text2324736937",
                max: 0,
                min: 0,
                name: "name",
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
        const collection = app.findCollectionByNameOrId("pbc_183765882");

        unmarshal(
            {
                indexes: [],
            },
            collection,
        );

        collection.fields.removeById("text2077450625");
        collection.fields.removeById("text1579384326");
        collection.fields.removeById("text2324736937");

        // restore key
        collection.fields.addAt(
            12,
            new Field({
                autogeneratePattern: "",
                hidden: false,
                id: "text2324736937",
                max: 0,
                min: 0,
                name: "key",
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
);
