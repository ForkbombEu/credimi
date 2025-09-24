/// <reference path="../pb_data/types.d.ts" />
migrate(
    (app) => {
        app.db()
            .newQuery(
                `
    UPDATE use_cases_verifications
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
    UPDATE use_cases_verifications
    SET canonified_name = NULL;
  `,
            )
            .execute();
    },
);
