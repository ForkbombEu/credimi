/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        app.db()
            .newQuery(
                `
    UPDATE credential_issuers
    SET canonified_name = id
    WHERE canonified_name IS NULL OR canonified_name = '';
  `,
            )
            .execute();
    },
    (app) => {
        app.db()
            .newQuery(
                `
    UPDATE credential_issuers
    SET canonified_name = NULL;
  `,
            )
            .execute();
    },
);
