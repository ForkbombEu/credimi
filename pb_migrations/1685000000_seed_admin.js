// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/// <reference path="../pb_data/types.d.ts" />
// @ts-check

const ADMIN_EMAIL = "admin@example.org";
const SUPERUSERS = "_superusers";
// Set only by local dev (`make dev`) and test fixtures. Deployments leave it
// unset and create their first superuser through the PocketBase installer.
const SEED_PASSWORD_ENV = "CREDIMI_SEED_SUPERUSER_PASSWORD";

migrate(
    (app) => {
        const password = $os.getenv(SEED_PASSWORD_ENV);
        if (!password) {
            return;
        }
        try {
            app.findAuthRecordByEmail(SUPERUSERS, ADMIN_EMAIL);
        } catch {
            const superusers = app.findCollectionByNameOrId(SUPERUSERS);
            const admin = new Record(superusers);
            admin.setEmail(ADMIN_EMAIL);
            admin.setPassword(password);
            app.save(admin);
        }
    },
    (app) => {
        try {
            const admin = app.findAuthRecordByEmail(SUPERUSERS, ADMIN_EMAIL);
            app.delete(admin);
        } catch {
            // nothing was seeded
        }
    }
);
