// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// @ts-check

/// <reference path="../pb_data/types.d.ts" />
/** @typedef {import('./utils.js')} Utils */
/** @typedef {import('./auditLogger.js')} AuditLogger */

/* CRUD hooks for orgJoinRequest */

onRecordCreateRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    // 0 - Base check

    if (!e.record) throw utils.createMissingDataError("orgJoinRequest");

    // 1 – Check if a membership already exists for that user and org

    const organizationId = e.record.getString("organization");
    const userId = e.record.getString("user");

    const authorization = utils.findFirstRecordByFilter(
        "orgAuthorizations",
        `organization.id = "${organizationId}" && user.id = "${userId}"`
    );

    if (authorization)
        throw new BadRequestError(utils.errors.user_is_already_member);

    // 2 - Set default values for the record

    e.record?.set("status", "pending");
    e.record?.set("reminders", 0);

    // 3 - Do the backend logic

    e.next();

    // 4 - Audit log the fact

    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    auditLogger(e).info(
        "Created membership request",
        "organizationId",
        e.record?.get("organization"),
        "requestId",
        e.record?.id
    );

    // 5 - Notify the admins

    const organization = utils.getExpanded(e.record, "organization");
    if (!organization)
        throw utils.createMissingDataError("organization of orgJoinRequest");
    const user = utils.getExpanded(e.record, "user");
    if (!user) throw utils.createMissingDataError("user of orgJoinRequest");

    const recipients = utils.getOrganizationAdminsAddresses(organizationId);

    for (const adminAddress of recipients) {
        const email = utils.renderEmail("membership-request-new", {
            OrganizationName: organization.getString("name"),
            Admin: adminAddress.name ?? "Admin",
            UserName: user.getString("name"),
            DashboardLink: utils.getOrganizationMembersPageUrl(organizationId),
            AppName: utils.getAppName(),
            AppLogo: utils.getAppLogoUrl()
        });

        const res = utils.sendEmail({
            to: adminAddress,
            ...email,
        });

        if (res instanceof Error) {
            console.error("Email send error");
        }
    }
}, "orgJoinRequests");

//

onRecordUpdateRequest((e) => {
    // `orgJoinRequests` can be updated only by admins of the org
    // This is expressed inside API rules

    e.next();

    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (!e.record) throw utils.createMissingDataError("orgJoinRequest");

    const status = e.record.getString("status");

    // 0 - Audit log

    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    auditLogger(e).info(
        "Updated membership request",
        "organizationId",
        e.record.getString("organization"),
        "status",
        status,
        "requestId",
        e.record.id
    );

    // 1 - Create orgAuthorization after if membership request is accepted

    if (status == "accepted") {
        const orgAuthorizationsCollection =
            $app.findCollectionByNameOrId("orgAuthorizations");
        if (!orgAuthorizationsCollection)
            throw utils.createMissingDataError("orgAuthorizationsCollection");

        const organizationId = e.record.getString("organization");
        const userId = e.record.getString("user");

        const memberRole = utils.getRoleByName("member");
        const roleId = memberRole?.id;

        const record = new Record(orgAuthorizationsCollection, {
            user: userId,
            organization: organizationId,
            role: roleId,
        });

        $app.save(record);
    }

    // 2 - Notifying user about the change

    if (status == "accepted" || status == "rejected") {
        const organization = utils.getExpanded(e.record, "organization");
        if (!organization)
            throw utils.createMissingDataError(
                "organization of orgJoinRequest"
            );
        const user = utils.getExpanded(e.record, "user");
        if (!user) throw utils.createMissingDataError("user of orgJoinRequest");

        const OrganizationName = organization.getString("name");
        const userAddress = utils.getUserEmailAddressData(user);

        const successEmail = utils.renderEmail("membership-request-accepted", {
            OrganizationName,
            UserName: userAddress.name ?? "User",
            DashboardLink: utils.getOrganizationPageUrl(organization.id),
            AppName: utils.getAppName(),
            AppLogo: utils.getAppLogoUrl()
        });

        const rejectEmail = utils.renderEmail("membership-request-declined", {
            OrganizationName,
            UserName: userAddress.name ?? "User",
            DashboardLink: utils.getAppUrl() + "/my/organizations",
            AppName: utils.getAppName(),
            AppLogo: utils.getAppLogoUrl()
        });

        const emailContent = status == "accepted" ? successEmail : rejectEmail;
        utils.sendEmail({ to: [userAddress], ...emailContent });
    }

    // 3 - Finally, deleting the record

    if (status == "accepted" || status == "rejected") {
        $app.delete(e.record);
    }
}, "orgJoinRequests");

//

onRecordDeleteRequest((e) => {
    // `orgJoinRequests` can be deleted only by the creator of the request
    // This is expressed inside API rules

    e.next();

    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    auditLogger(e).info(
        "Deleted membership request",
        "organizationId",
        e.record?.getString("organization"),
        "status",
        e.record?.getString("status"),
        "requestId",
        e.record?.id
    );
}, "orgJoinRequests");

//

cronAdd("remind admins about join requests", "0 9 * * 1", () => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    const organizations = utils.findRecordsByFilter(
        "organizations",
        "id != null"
    );

    organizations
        .map((organization) => ({
            organization,
            requests: utils
                .findRecordsByFilter(
                    "orgJoinRequests",
                    `organization.id = "${organization.id}"`
                )
                .filter((r) => r.get("status") == "pending"),
        }))
        .filter(({ requests }) => requests.length > 0)
        .forEach(({ organization, requests }) => {
            const organizationId = organization.id;
            const OrganizationName = organization.get("name");

            const recipients =
                utils.getOrganizationAdminsAddresses(organizationId);

            for (const recipient of recipients) {
                const email = utils.renderEmail("membership-request-pending", {
                    OrganizationName,
                    DashboardLink:
                        utils.getOrganizationMembersPageUrl(organizationId),
                    Admin: recipient.name ?? "Admin",
                    PendingNumber: requests.length.toString(),
                    AppName: utils.getAppName(),
                    AppLogo: utils.getAppLogoUrl()
                });

                const res = utils.sendEmail({
                    to: recipient,
                    ...email,
                });
                if (res instanceof Error) {
                    console.error("Email send error");
                }
            }
        });
});
