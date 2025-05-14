// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// @ts-check

/// <reference path="../pb_data/types.d.ts" />
/** @typedef {import('./utils.js')} Utils */

onRecordCreateRequest(
    (e) => {
        /** @type {Utils} */
        const utils = require(`${__hooks}/utils.js`);

        utils.skipIfAdmin(e);

        e.record?.set("owner", utils.getRequestingUserOrganization(e));
        e.next();
    },
    "custom_checks",
    "wallets",
    "verifiers"
);

onRecordUpdateRequest(
    (e) => {
        /** @type {Utils} */
        const utils = require(`${__hooks}/utils.js`);

        utils.skipIfAdmin(e);

        e.record?.set("owner", utils.getRequestingUserOrganization(e));
        e.next();
    },
    "custom_checks",
    "wallets",
    "verifiers"
);
