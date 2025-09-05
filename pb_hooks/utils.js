// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// @ts-check

/// <reference path="../pb_data/types.d.ts" />
/// <reference path="./ambient.d.ts" />

/** @typedef {import("../webapp/src/modules/pocketbase/types").OrgRolesRecord} OrgRole */
/** @typedef {import("../webapp/src/modules/pocketbase/types").OrgAuthorizationsRecord} OrgAuthorization */
/** @typedef {import("../webapp/src/modules/pocketbase/types").UsersRecord} User */

/** @typedef {MailerMessage["to"][number]} Address */

/** @typedef {import('./auditLogger.js')} AuditLogger */

//

/* -- Error codes -- */

const errors = {
    not_authorized: "not_authorized",
    missing_data: "missing_data",
    user_not_logged: "user_not_logged",

    cant_create_an_authorization_for_yourself:
        "cant_create_an_authorization_for_yourself",

    cant_edit_last_owner_role: "cant_edit_last_owner_role",
    cant_delete_last_owner_role: "cant_delete_last_owner_role",

    cant_create_role_higher_than_or_equal_to_yours:
        "cant_create_role_higher_than_or_equal_to_yours",
    cant_edit_role_higher_than_or_equal_to_yours:
        "cant_edit_role_higher_than_or_equal_to_yours",
    cant_delete_role_higher_than_or_equal_to_yours:
        "cant_delete_role_higher_than_or_equal_to_yours",

    user_is_already_member: "user_is_already_member",
};

/* -- RBAC Utils -- */

/**
 * @param {string} name
 * @returns {RecordModel<OrgRole> | undefined}
 */
function getRoleByName(name) {
    try {
        return $app.findFirstRecordByData("orgRoles", "name", name);
    } catch {
        return undefined;
    }
}

/**
 * @param {core.Record} role
 * @returns {number}
 */
function getRoleLevel(role) {
    return role.get("level");
}

/**
 * @param {core.Record} orgAuthorization
 */
function isLastOwnerAuthorization(orgAuthorization) {
    const organizationId = orgAuthorization.get("organization");
    const roleId = orgAuthorization.get("role");
    const ownerRoleId = getRoleByName("owner")?.id;

    if (roleId !== ownerRoleId) return false;

    const ownerAuthorizations = findRecordsByFilter(
        "orgAuthorizations",
        `organization="${organizationId}" && role="${ownerRoleId}"`
    );

    return ownerAuthorizations.length == 1;
}

/**
 * @param {string} userId
 * @param {string} organizationId
 * @param {core.App | excludeHooks<PocketBase> } [app= $app]
 * @returns {RecordModel<OrgRole> | undefined}
 */
function getUserRole(userId, organizationId, app = $app) {
    const authorization = findFirstRecordByFilter(
        "orgAuthorizations",
        `user = "${userId}" && organization = "${organizationId}"`,
        app
    );
    if (!authorization) return undefined;
    return getExpanded(authorization, "role", app);
}

/**
 *
 * @param {core.RecordRequestEvent} e
 */
function getUserContextInOrgAuthorizationHookEvent(e) {
    const userId = getUserFromContext(e)?.id;

    /** @type {string | undefined} */
    const organizationId = e.record?.get("organization");

    if (!userId || !organizationId)
        throw createMissingDataError("requestingUserId", "organizationId");

    const isSelf = userId === e.record?.get("user");

    const userRole = getUserRole(userId, organizationId);
    if (!userRole) throw createMissingDataError("requestingUserRole");

    const userRoleLevel = getRoleLevel(userRole);

    return { userId, userRole, isSelf, userRoleLevel };
}

/* -- Pocketbase utils -- */

/**
 * @param {core.RequestEvent} e
 * @returns {RecordModel<User> | undefined}
 */
function getUserFromContext(e) {
    return e.auth;
}

/**
 * @param {string} collection
 * @param {string} filter
 * @param {core.App | excludeHooks<PocketBase> } [app= $app]
 * @returns {Array<core.Record>}
 */
function findRecordsByFilter(collection, filter, app = $app) {
    return app
        .findRecordsByFilter(collection, filter, "", 0, 0)
        .filter((v) => v != undefined);
}

/**
 * @param {string} collection
 * @param {string} filter
 * @param {core.App | excludeHooks<PocketBase> } [app= $app]
 */
function findFirstRecordByFilter(collection, filter, app = $app) {
    try {
        return app.findFirstRecordByFilter(collection, filter);
    } catch {
        return undefined;
    }
}

/**
 * @param {core.Record} record
 * @param {string} key
 * @param {core.App | excludeHooks<PocketBase> } [app= $app]
 * @returns {core.Record | undefined}
 */
function getExpanded(record, key, app = $app) {
    try {
        // @ts-ignore
        app.expandRecord(record, [key], null);
        return record.expandedOne(key);
    } catch (e) {
        return undefined;
    }
}

/**
 * @param {core.RequestEvent} e
 */
function isAdminContext(e) {
    return Boolean(e.hasSuperuserAuth());
}

/**
 * @param {string[]} args
 */
function createMissingDataError(...args) {
    return new BadRequestError(errors.missing_data + ": " + args.join(", "));
}

/**
 * @param {core.Record} user
 * @returns {Address}
 */
function getUserEmailAddressData(user) {
    /** @type {string} */
    const name = user.get("name");
    /** @type {string} */
    const address = user.get("email");
    if (!name || !address)
        throw createMissingDataError("userEmail", "userName");
    return {
        name,
        address,
    };
}

/**
 * @param {{from?: Address, to: Address[] | Address, subject: string, html: string}} data
 * @return {Error | undefined}
 */
function sendEmail(data) {
    try {
        const message = new MailerMessage({
            from: {
                address:
                    data.from?.address ?? $app.settings().meta.senderAddress,
                name: data.from?.name ?? $app.settings().meta.senderName,
            },
            to: Array.isArray(data.to) ? data.to : [data.to],
            subject: data.subject,
            html: data.html,
        });

        $app.newMailClient().send(message);
    } catch (e) {
        return e;
    }
}

/**
 * @param {string} string
 * @returns {string}
 */
function removeTrailingSlash(string) {
    if (string.endsWith("/")) return string.slice(0, -1);
    else return string;
}

/**
 * @param {string} organizationId
 * @param {core.App | excludeHooks<PocketBase> } [app= $app]
 * @returns {Address[]}
 */
function getOrganizationAdminsAddresses(organizationId, app = $app) {
    const recipients = findRecordsByFilter(
        "orgAuthorizations",
        `organization.id = "${organizationId}" && ( role.name = "admin" || role.name = "owner" )`,
        app
    );

    return recipients
        .map((r) => getExpanded(r, "user"))
        .filter((u) => u != undefined)
        .map((u) => getUserEmailAddressData(u));
}

/** @returns {string} */
function getAppUrl() {
    return removeTrailingSlash($app.settings().meta.appURL);
}

/** @returns {string} */
function getAppName() {
    return $app.settings().meta.appName;
}

/**
 * @returns {string}
 */
function getAppLogoUrl() {
    return getAppUrl() + `/logos/${getAppName().toLowerCase()}_logo-transp_emblem.png`
}

/**
 * @param {string} organizationId
 * @returns {string}
 */
function getOrganizationPageUrl(organizationId) {
    return `${getAppUrl()}/my/organizations/${organizationId}`;
}

/**
 * @param {string} organizationId
 * @returns {string}
 */
function getOrganizationMembersPageUrl(organizationId) {
    return `${getOrganizationPageUrl(organizationId)}/members`;
}

/**
 * @param {core.RequestEvent} e
 */
function runOrganizationInviteEndpointChecks(e) {
    /** @type {{inviteId: string | undefined}} */
    // @ts-ignore
    const { inviteId } = e.requestInfo().body;
    if (!inviteId || typeof inviteId != "string")
        throw createMissingDataError("inviteId");

    const userId = getUserFromContext(e)?.id;
    if (!userId) throw createMissingDataError("userId");

    const invite = findFirstRecordByFilter("org_invites", `id = "${inviteId}"`);
    if (!invite) throw createMissingDataError("organization invite");

    const isOwner = invite.get("user") == userId;
    if (!isOwner) throw new ForbiddenError();

    return { userId, invite, isOwner };
}

/**
 *
 * @param {core.RecordRequestEvent} event
 * @param {string[]} fields
 */
function getRecordUpdateEventDiff(event, fields = []) {
    const updatedRecord = event.record;
    const originalRecord = event.record?.original();
    if (!updatedRecord || !originalRecord)
        throw createMissingDataError("updated record");

    if (fields.length == 0)
        fields = getCollectionFields(updatedRecord.collection());

    return fields
        .map((f) => ({
            field: f,
            newValue: updatedRecord.get(f),
            oldValue: originalRecord.get(f),
        }))
        .filter((d) => d.newValue != d.oldValue);
}

/**
 * @param {core.Collection} collection
 */
function getCollectionFields(collection) {
    return collection.fields.map((f) => f?.name).filter((n) => n != undefined);
}

//

/** @type {EmailRenderer} */
const renderEmail = (name, data) => {
    const emailPath = $filepath.join(
        __hooks,
        "..",
        "webapp",
        "static",
        "emails",
        `${String(name)}.html`
    );
    const html = $template
        .loadFiles(emailPath)
        .render(data)
        .replace("@WL", "{{Weblink}}")
        .replace("@US", "{{{unsubscribe}}}");
    const subject = html.match(/<title>(.*?)<\/title>/)?.at(1) ?? "";
    return {
        html,
        subject,
    };
};

/**
 *
 * @param {core.RecordRequestEvent} event
 * @param {string[]} fields
 */
function getRecordUpdateEventDiff(event, fields = []) {
    const updatedRecord = event.record;
    const originalRecord = event.record?.original();
    if (!updatedRecord || !originalRecord)
        throw createMissingDataError("updated record");

    if (fields.length == 0)
        fields = getCollectionFields(updatedRecord.collection());

    return fields
        .map((f) => ({
            field: f,
            newValue: updatedRecord.get(f),
            oldValue: originalRecord.get(f),
        }))
        .filter((d) => d.newValue != d.oldValue);
}

/**
 * @param {core.Collection} collection
 */
function getCollectionFields(collection) {
    return collection.fields.map((f) => f?.name).filter((n) => n != undefined);
}

/**
 *
 * @param { core.RequestEvent } e
 * @returns { RecordModel<User> | undefined }
 */
function getRequestAgent(e) {
    /** @type {RecordModel<User> | undefined} */
    let agent = undefined;

    const adminContext = isAdminContext(e);
    const user = getUserFromContext(e);

    if (adminContext) {
        agent = e.auth;
    } else if (user) {
        agent = user;
    }

    return agent;
}

/**
 *
 * @param { core.RequestEvent } e
 * @returns { string | undefined }
 */
function getRequestAgentName(e) {
    const agent = getRequestAgent(e);
    if (!agent) return undefined;

    if ("getString" in agent) {
        return agent.getString("name");
    }
}

/**
 * @param {core.Record} record
 * @param {string} field
 * @returns {string}
 */
function getFileKey(record, field) {
    return record.baseFilesPath() + "/" + record.getString(field);
}

/**
 *
 * @param {core.Record} record
 * @param {string} field
 * @returns {filesystem.File | undefined}
 */
function copyFile(record, field) {
    let fsys, fileBlob;
    let file;
    try {
        fsys = $app.newFilesystem();
        fileBlob = fsys.getFile(getFileKey(record, field));
        file = $filesystem.fileFromBytes(
            toString(fileBlob),
            record.getString(field)
        );
    } catch (e) {
        console.error(e);
    } finally {
        fileBlob?.close();
        fsys?.close();
    }
    return file;
}

/**
 *
 * @param {core.RecordRequestEvent} e
 * @param {string} organizationId
 * @param {string} userId
 * @param {string} [organizationName]
 */
function createOwnerRoleForOrganization(
    e,
    organizationId,
    userId,
    organizationName = organizationId
) {
    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    //

    const ownerRole = getRoleByName("owner");
    const ownerRoleId = ownerRole?.id;

    const collection = $app.findCollectionByNameOrId("orgAuthorizations");
    const record = new Record(collection, {
        organization: organizationId,
        role: ownerRoleId,
        user: userId,
    });
    $app.save(record);

    auditLogger(e).info(
        "Created owner role for organization",
        "organizationId",
        e.record?.id,
        "organizationName",
        organizationName,
        "userId",
        userId
    );
}

/**
 * @param {core.RequestEvent} e
 * @returns {string | undefined}
 */
function getRequestingUserOrganization(e) {
    if (isAdminContext(e)) return undefined;

    const userId = getUserFromContext(e)?.id;
    if (!userId) return undefined;

    const orgAuth = findFirstRecordByFilter(
        "orgAuthorizations",
        `user = "${userId}"`
    );
    if (!orgAuth) return undefined;
    return orgAuth.get("organization");
}

/**
 * @param {core.RequestEvent} e
 */
function skipIfAdmin(e) {
    if (isAdminContext(e)) e.next();
}

//

module.exports = {
    getUserFromContext,
    getRoleByName,
    findRecordsByFilter,
    isLastOwnerAuthorization,
    findFirstRecordByFilter,
    getUserRole,
    getExpanded,
    isAdminContext,
    getRoleLevel,
    createMissingDataError,
    getUserContextInOrgAuthorizationHookEvent,
    getUserEmailAddressData,
    sendEmail,
    removeTrailingSlash,
    getOrganizationAdminsAddresses,
    getOrganizationMembersPageUrl,
    getAppUrl,
    getAppLogoUrl,
    getAppName,
    runOrganizationInviteEndpointChecks,
    renderEmail,
    getOrganizationPageUrl,
    getRecordUpdateEventDiff,
    getRequestAgent,
    getRequestAgentName,
    copyFile,
    createOwnerRoleForOrganization,
    getRequestingUserOrganization,
    skipIfAdmin,
    errors,
};
