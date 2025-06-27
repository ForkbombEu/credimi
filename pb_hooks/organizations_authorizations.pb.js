// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// @ts-check

/// <reference path="../pb_data/types.d.ts" />
/** @typedef {import('./utils.js')} Utils */
/** @typedef {import('./auditLogger.js')} AuditLogger */
/** @typedef {import("../webapp/src/modules/pocketbase/types").OrgAuthorizationsRecord} OrgAuthorization */
/** @typedef {import("../webapp/src/modules/pocketbase/types").OrgRolesResponse} OrgRole */

/**
 * INDEX
 * - guard hooks (protecting orgAuthorizations from invalid CRUD operations)
 * - audit + email hooks
 */

/* Guard hooks */

// [CREATE] Cannot create an authorization with a level higher than or equal to your permissions

onRecordCreateRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (utils.isAdminContext(e)) e.next();

    const { isSelf, userRoleLevel } =
        utils.getUserContextInOrgAuthorizationHookEvent(e);

    if (isSelf)
        throw new BadRequestError(
            utils.errors.cant_create_an_authorization_for_yourself
        );

    // Getting requested role level

    if (!e.record) throw utils.createMissingDataError("orgAuthorization");
    const requestedRole = utils.getExpanded(e.record, "role");
    if (!requestedRole) throw utils.createMissingDataError("requestedRole");
    const requestedRoleLevel = utils.getRoleLevel(requestedRole);

    // Matching

    if (requestedRoleLevel <= userRoleLevel) {
        throw new BadRequestError(
            utils.errors.cant_create_role_higher_than_or_equal_to_yours
        );
    }

    e.next();
}, "orgAuthorizations");

// [UPDATE] Cannot update to/from a role higher than the user

onRecordUpdateRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (utils.isAdminContext(e)) e.next();

    const { isSelf, userRoleLevel: requestingUserRoleLevel } =
        utils.getUserContextInOrgAuthorizationHookEvent(e);

    // Getting role before edit (unmodified)

    const originalAuthorization = e.record?.original();
    if (!originalAuthorization)
        throw utils.createMissingDataError("originalAuthorization");

    const originalRole = utils.getExpanded(originalAuthorization, "role");
    if (!originalRole) throw utils.createMissingDataError("previousRole");

    const originalRoleLevel = utils.getRoleLevel(originalRole);

    // First check

    if (originalRoleLevel <= requestingUserRoleLevel && !isSelf)
        throw new ForbiddenError(
            utils.errors.cant_edit_role_higher_than_or_equal_to_yours
        );

    // Getting requested role

    /** @type {Partial<OrgAuthorization>} */
    const { role: newRoleId } = e.requestInfo().body;

    if (!newRoleId) throw utils.createMissingDataError("newRoleId");
    const newRole = $app.findRecordById("orgRoles", newRoleId);

    const newRoleLevel = utils.getRoleLevel(newRole);

    // Second check

    if (newRoleLevel <= requestingUserRoleLevel)
        throw new ForbiddenError(
            utils.errors.cant_edit_role_higher_than_or_equal_to_yours
        );

    e.next();
}, "orgAuthorizations");

// [DELETE] Cannot delete an authorization with a level higher than or equal to yours

onRecordDeleteRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (utils.isAdminContext(e)) e.next();

    const { isSelf, userRoleLevel: requestingUserRoleLevel } =
        utils.getUserContextInOrgAuthorizationHookEvent(e);

    // If user requests delete for itself, it's fine
    if (isSelf) e.next();

    // Getting role of authorization to delete

    if (!e.record) throw utils.createMissingDataError("originalAuthorization");

    const roleToDelete = utils.getExpanded(e.record, "role");
    if (!roleToDelete) throw utils.createMissingDataError("roleToDelete");

    const roleToDeleteLevel = utils.getRoleLevel(roleToDelete);

    // Comparing levels

    if (roleToDeleteLevel <= requestingUserRoleLevel)
        throw new ForbiddenError(
            utils.errors.cant_delete_role_higher_than_or_equal_to_yours
        );

    e.next();
}, "orgAuthorizations");

// [DELETE] Cannot delete last owner role

onRecordDeleteRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (utils.isAdminContext(e)) e.next();

    if (e.record && utils.isLastOwnerAuthorization(e.record)) {
        throw new BadRequestError(utils.errors.cant_edit_last_owner_role);
    }

    e.next();
}, "orgAuthorizations");

// [UPDATE] Cannot edit last owner role

onRecordUpdateRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (utils.isAdminContext(e)) e.next();

    const originalRecord = e.record?.original();
    // e.record is already the "modified" version, so it is not a "owner" role anymore
    // to check if it's the last one, we need to get the "original" record

    if (originalRecord && utils.isLastOwnerAuthorization(originalRecord)) {
        throw new BadRequestError(utils.errors.cant_delete_last_owner_role);
    }

    e.next();
}, "orgAuthorizations");

/* Audit + Email hooks */

onRecordCreateRequest((e) => {
    e.next();

    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (!e.record) throw utils.createMissingDataError("orgAuthorization");

    const organization = utils.getExpanded(e.record, "organization");
    const user = utils.getExpanded(e.record, "user");
    const role = utils.getExpanded(e.record, "role");

    auditLogger(e).info(
        "Created organization authorization",
        "organizationId",
        organization?.id,
        "organizationName",
        organization?.get("name"),
        "userId",
        user?.id,
        "userName",
        user?.get("name"),
        "roleId",
        role?.id,
        "roleName",
        role?.get("name")
    );
}, "orgAuthorizations");

onRecordUpdateRequest((e) => {
    e.next();

    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (!e.record) throw utils.createMissingDataError("orgAuthorization");

    const organization = utils.getExpanded(e.record, "organization");
    if (!organization) throw utils.createMissingDataError("organization");
    const OrganizationName = organization.get("name");

    const user = utils.getExpanded(e.record, "user");
    if (!user) throw utils.createMissingDataError("user of orgAuthorization");
    const UserName = user.getString("name");

    const previousRole = utils.getExpanded(e.record.original(), "role");
    const role = utils.getExpanded(e.record, "role");
    if (!role) throw utils.createMissingDataError("role");

    const adminName = e.auth?.getString("name");
    if (!adminName) throw utils.createMissingDataError("adminName");

    auditLogger(e).info(
        "Updated organization authorization",
        "organizationId",
        organization.id,
        "organizationName",
        OrganizationName,
        "userId",
        user.id,
        "userName",
        UserName,
        "previousRoleId",
        previousRole?.id,
        "previousRoleName",
        previousRole?.get("name"),
        "newRoleId",
        role.id,
        "newRoleName",
        role.getString("name")
    );

    const email = utils.renderEmail("role-change", {
        OrganizationName,
        DashboardLink: utils.getOrganizationPageUrl(organization.id),
        UserName,
        Admin: adminName,
        Membership: role.getString("name"),
        AppName: utils.getAppName(),
        AppLogo: utils.getAppLogoUrl()
    });

    const res = utils.sendEmail({
        to: utils.getUserEmailAddressData(user),
        ...email,
    });

    if (res instanceof Error) {
        console.error(res);
    }
}, "orgAuthorizations");

onRecordDeleteRequest((e) => {
    e.next();

    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (!e.record) throw utils.createMissingDataError("orgAuthorization");

    const record = e.record.original();

    const organization = utils.getExpanded(record, "organization");
    const OrganizationName = organization?.get("name");

    const user = utils.getExpanded(record, "user");
    const role = utils.getExpanded(record, "role");
    if (!user) throw utils.createMissingDataError("user of orgAuthorization");
    if (!role) throw utils.createMissingDataError("role of orgAuthorization");

    auditLogger(e).info(
        "Deleted organization authorization",
        "organizationId",
        organization?.id,
        "organizationName",
        OrganizationName,
        "userId",
        user.id,
        "userName",
        user.get("name"),
        "roleId",
        role.id,
        "roleName",
        role.get("name")
    );

    const email = utils.renderEmail("member-removal", {
        OrganizationName,
        DashboardLink: utils.getAppUrl(),
        UserName: user.getString("name"),
        AppName: utils.getAppName(),
        AppLogo: utils.getAppLogoUrl()
    });

    const res = utils.sendEmail({
        to: utils.getUserEmailAddressData(user),
        ...email,
    });
    if (res instanceof Error) console.error(res);
}, "orgAuthorizations");
