// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/// <reference path="../pb_data/types.d.ts" />
// @ts-check

const ADMIN_EMAIL = "admin@example.org";
const DEFAULT_PASSWORD = "adminadmin";
const SUPERUSERS = "_superusers";
const SEED_PASSWORD_ENV = "CREDIMI_SEED_SUPERUSER_PASSWORD";

// Older deployments got admin@example.org with a public default password from
// 1685000000_seed_admin.js. Remove that credential unless this is a dev/test
// instance that seeds the admin on purpose.
migrate(
    (app) => {
        if ($os.getenv(SEED_PASSWORD_ENV)) {
            return;
        }
        let admin;
        try {
            admin = app.findAuthRecordByEmail(SUPERUSERS, ADMIN_EMAIL);
        } catch {
            return;
        }
        if (!admin.validatePassword(DEFAULT_PASSWORD)) {
            return;
        }
        if (app.countRecords(SUPERUSERS) > 1) {
            app.delete(admin);
            return;
        }
        // PocketBase refuses to delete the only superuser: lock the account
        // instead. Recover with `credimi superuser upsert <email> <password>`.
        admin.setPassword($security.randomString(40));
        app.save(admin);
    },
    () => {
        // the default credential must not come back
    }
);
