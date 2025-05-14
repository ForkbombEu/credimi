// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// @ts-check

/// <reference path="../pb_data/types.d.ts" />
/** @typedef {import('./utils.js')} Utils */

// TODO - Add organization ownership

onRecordCreateRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (!utils.isAdminContext(e)) {
        e.record?.set("owner", e.auth?.id);
    }

    e.next();
}, "custom_checks");

onRecordUpdateRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    // TODO - If possible, instead of setting the owner again, just remove the field from the request body
    if (!utils.isAdminContext(e)) {
        e.record?.set("owner", e.auth?.id);
    }

    e.next();
}, "custom_checks");
