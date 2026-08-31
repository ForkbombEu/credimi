/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_2980015441");

        // update field: raise the FCAF report PDF size limit
        // (maxSize 0 falls back to the 5 MiB PocketBase default)
        collection.fields.addAt(
            15,
            new Field({
                hidden: false,
                id: "file_fcaf_report_pdf",
                maxSize: 52428800,
                maxSelect: 1,
                mimeTypes: ["application/pdf"],
                name: "fcaf_report_pdf",
                presentable: false,
                protected: false,
                required: false,
                system: false,
                thumbs: [],
                type: "file",
            }),
        );

        return app.save(collection);
    },
    (app) => {
        const collection = app.findCollectionByNameOrId("pbc_2980015441");

        // update field: restore the default FCAF report PDF size limit
        collection.fields.addAt(
            15,
            new Field({
                hidden: false,
                id: "file_fcaf_report_pdf",
                maxSize: 0,
                maxSelect: 1,
                mimeTypes: ["application/pdf"],
                name: "fcaf_report_pdf",
                presentable: false,
                protected: false,
                required: false,
                system: false,
                thumbs: [],
                type: "file",
            }),
        );

        return app.save(collection);
    },
);
