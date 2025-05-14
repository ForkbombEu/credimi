// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// @ts-check

/// <reference path="../pb_data/types.d.ts" />
/** @typedef {import('./utils.js')} Utils */
/** @typedef {import('./auditLogger.js')} AuditLogger */

/**
 * INDEX
 * - Routes
 * - Business logic hooks
 * - Audit hooks
 * - Email hooks
 */

/* Routes */

routerAdd("POST", "/organizations/verify-user-membership", (e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);
    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    const userId = utils.getUserFromContext(e)?.id;

    /** @type {string | undefined} */
    const organizationId = e.requestInfo().body["organizationId"];
    if (!organizationId)
        throw utils.createMissingDataError("organizationId", "roles");

    try {
        $app.findFirstRecordByFilter(
            "orgAuthorizations",
            `organization="${organizationId}" && user="${userId}"`
        );
        return e.json(200, { isMember: true });
    } catch {
        auditLogger(e).info(
            "request_from_user_not_member",
            "organizationId",
            organizationId
        );
        return e.json(200, { isMember: false });
    }
});

routerAdd("POST", "/organizations/verify-user-role", (e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    const userId = utils.getUserFromContext(e)?.id;

    /** @type {{organizationId: string, roles: string[]}}*/
    // @ts-ignore
    const { organizationId, roles } = e.requestInfo().body;
    if (!organizationId || !roles || roles.length === 0)
        throw utils.createMissingDataError("organizationId", "roles");

    const roleFilter = `( ${roles
        .map((r) => `role.name="${r}"`)
        .join(" || ")} )`;

    try {
        $app.findFirstRecordByFilter(
            "orgAuthorizations",
            `organization="${organizationId}" && user="${userId}" && ${roleFilter}`
        );
        return e.json(200, { hasRole: true });
    } catch {
        return e.json(200, { hasRole: false });
    }
});

/* Business logic hooks */

// Create organization when user is created

onRecordCreateRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);
    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    const user = e.record;
    if (!user) throw utils.createMissingDataError("user");

    $app.runInTransaction((txApp) => {
        // Creating main organization

        const organizationCollection =
            txApp.findCollectionByNameOrId("organizations");
        const organizationName = user.getString("email").split("@")[0];
        const organization = new Record(organizationCollection, {
            name: organizationName,
        });
        txApp.save(organization);

        auditLogger(e, txApp).info(
            "Created organization",
            "organizationId",
            organization.id,
            "organizationName",
            organizationName,
            "user",
            user.getString("email")
        );

        // Creating info

        const organizationInfoCollection =
            txApp.findCollectionByNameOrId("organization_info");
        const organizationInfo = new Record(organizationInfoCollection, {
            name: organizationName,
            owner: user.id,
        });
        txApp.save(organizationInfo);

        // Creating owner role

        const ownerRole = utils.getRoleByName("owner");
        const ownerRoleId = ownerRole?.id;

        const authorizationCollection =
            txApp.findCollectionByNameOrId("orgAuthorizations");
        const record = new Record(authorizationCollection, {
            organization: organization.id,
            role: ownerRoleId,
            user: user.id,
        });
        txApp.save(record);

        auditLogger(e, txApp).info(
            "Created owner role for organization",
            "organizationId",
            organization.id,
            "organizationName",
            organizationName,
            "user",
            user.getString("email")
        );
    });
}, "users");
