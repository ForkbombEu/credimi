/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("aako88kt3br4npt");

        // clear indexes so PB doesn’t reapply old ones
        unmarshal(
            {
                indexes: [],
            },
            collection,
        );

        // add canonified_name field
        collection.fields.addAt(
            2,
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
        const collection = app.findCollectionByNameOrId("aako88kt3br4npt");

        unmarshal(
            {
                indexes: [],
            },
            collection,
        );

        collection.fields.removeById("text2077450625");

        return app.save(collection);
    },
);
