// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// @ts-check

/// <reference path="../pb_data/types.d.ts" />
/** @typedef {import('./utils.js')} Utils */

onRecordCreateRequest(
    (e) => {
        e.record?.set("imported", false);
        e.next();
    },
    "credential_issuers",
    "credentials"
);

onRecordUpdateRequest(
    (e) => {
        e.record?.set("imported", false);
        e.next();
    },
    "credential_issuers",
    "credentials"
);
