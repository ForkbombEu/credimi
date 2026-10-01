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

routerAdd(
    "POST",
    "/organizations/verify-user-membership",
    (e) => {
        /** @type {Utils} */
        const utils = require(`${__hooks}/utils.js`);
        /** @type {AuditLogger} */
        const auditLogger = require(`${__hooks}/auditLogger.js`);

        const userId = utils.getUserFromContext(e)?.id;
        if (!userId) throw new UnauthorizedError();

        /** @type {unknown} */
        const organizationId = e.requestInfo().body["organizationId"];
        if (!organizationId || typeof organizationId !== "string")
            throw utils.createMissingDataError("organizationId");

        const authorization = utils.findFirstRecordByFilter(
            "orgAuthorizations",
            "organization = {:organizationId} && user = {:userId}",
            { organizationId, userId }
        );
        if (authorization) return e.json(200, { isMember: true });

        auditLogger(e).info(
            "request_from_user_not_member",
            "organizationId",
            organizationId
        );
        return e.json(200, { isMember: false });
    },
    $apis.requireAuth()
);

routerAdd(
    "POST",
    "/organizations/verify-user-role",
    (e) => {
        /** @type {Utils} */
        const utils = require(`${__hooks}/utils.js`);

        const userId = utils.getUserFromContext(e)?.id;
        if (!userId) throw new UnauthorizedError();

        /** @type {{organizationId: unknown, roles: unknown}} */
        // @ts-ignore
        const { organizationId, roles } = e.requestInfo().body;
        if (
            !organizationId ||
            typeof organizationId !== "string" ||
            !Array.isArray(roles) ||
            roles.length === 0 ||
            roles.some((r) => typeof r !== "string")
        )
            throw utils.createMissingDataError("organizationId", "roles");

        // Compare role names in JS: request values never reach the filter.
        const roleName = utils.getUserRole(userId, organizationId)?.get("name");
        return e.json(200, {
            hasRole: typeof roleName === "string" && roles.includes(roleName),
        });
    },
    $apis.requireAuth()
);

/* Business logic hooks */

onRecordCreateRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);
    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    // 0 - Backend logic

    e.next();

    if (!e.record) throw utils.createMissingDataError("organization record");

    // 1 - Audit logging

    auditLogger(e).info(
        "Created organization",
        "organizationId",
        e.record?.id,
        "organizationName",
        e.record?.get("name")
    );

    if (utils.isAdminContext(e)) e.next();

    // 2 - Creating owner `orgAuthorization` for user that crated the org

    const user = utils.getUserFromContext(e);
    if (!user) throw utils.createMissingDataError("user creating organization");

    const organizationId = e.record.id;
    const organizationName = e.record.getString("name");

    utils.createOwnerRoleForOrganization(e, organizationId, user.id);

    // 3 - Finally, notifying user

    const userAddress = utils.getUserEmailAddressData(user);

    const emailData = utils.renderEmail("new-organization", {
        OrganizationName: organizationName,
        UserName: user.get("name") ?? "User",
        DashboardLink: utils.getOrganizationPageUrl(e.record.id),
        AppName: utils.getAppName(),
        AppLogo: utils.getAppLogoUrl()
    });

    const res = utils.sendEmail({
        to: userAddress,
        ...emailData,
    });
    if (res instanceof Error) {
        console.error(res);
    }
}, "organizations");
