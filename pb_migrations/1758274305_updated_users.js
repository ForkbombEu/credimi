/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        // backfill canonified_name with unique IDs
        app.db()
            .newQuery(
                `
    UPDATE users
    SET canonified_name = id
    WHERE canonified_name IS NULL OR canonified_name = '';
  `,
            )
            .execute();
    },
    (app) => {
        // rollback: clear the values
        app.db()
            .newQuery(
                `
    UPDATE users
    SET canonified_name = NULL;
  `,
            )
            .execute();
    },
);
