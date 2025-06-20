// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// @ts-check

/// <reference path="../pb_data/types.d.ts" />
/** @typedef {import('./utils.js')} Utils */
/** @typedef {import('./auditLogger.js')} AuditLogger */

//

onRecordCreateRequest((e) => {
    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    if (utils.isAdminContext(e)) e.next();

    //

    // Check if the provider is already claimed

    const provider = $app.findRecordById("services", e.record?.get("provider"));
    const isClaimed = Boolean(provider.getString("owner"));

    if (isClaimed) {
        throw new BadRequestError("This provider is already claimed.");
    }

    //

    const ownerId = utils.getUserFromContext(e)?.id;

    // Check if the user has already submitted a claim for this provider

    const providerClaims = $app.findRecordsByFilter(
        "provider_claims",
        [
            `provider = "${e.record?.get("provider")}"`,
            `owner = "${ownerId}"`,
        ].join(" && "),
        "created",
        1,
        0
    );

    if (providerClaims.length > 0) {
        throw new BadRequestError(
            "You have already submitted a claim for this provider."
        );
    }

    // Adding default values

    e.record?.set("status", "in_review");
    const owner = utils.getUserFromContext(e);
    e.record?.set("owner", owner?.id);

    auditLogger(e).info(
        "created_provider_claim",
        "providerId",
        e.record?.get("provider"),
        "providerName",
        e.record?.get("name"),
        "ownerId",
        owner?.id,
        "ownerName",
        owner?.getString("name")
    );

    //

    e.next();
}, "provider_claims");

//

onRecordUpdateRequest((e) => {
    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);

    const providerClaim = e.record;
    const originalProviderClaim = providerClaim?.original();

    // If the request is not from an admin, restore the original values
    if (!utils.isAdminContext(e)) {
        $app.runInTransaction(() => {
            ["status", "owner", "provider"].forEach((field) => {
                providerClaim?.set(field, originalProviderClaim?.get(field));
            });
        });
    }

    e.next();

    /**
     * After the update
     */

    // If the request is from an admin, and the status is "approved":
    // - Update the provider
    // - Create an organization if missing
    // - Delete the claim
    // - Notify the user

    if (
        utils.isAdminContext(e) &&
        providerClaim &&
        providerClaim.get("status") === "approved"
    ) {
        $app.runInTransaction((app) => {
            const ownerId = providerClaim.getString("owner");
            const logo = utils.copyFile(providerClaim, "logo");

            /**
             * Create an organization if missing
             */

            /** @type {string} */
            let organizationId;

            const orgAuthorizations = app
                .findRecordsByFilter(
                    "orgAuthorizations",
                    `user.id = "${ownerId}"`,
                    "",
                    0,
                    0
                )
                .filter((org) => org !== undefined);

            if (orgAuthorizations.length === 1) {
                organizationId = orgAuthorizations[0].getString("organization");
            } else if (orgAuthorizations.length > 1) {
                throw new BadRequestError(
                    "Multiple organizations found for the same user."
                );
            } else {
                const orgCollection =
                    app.findCollectionByNameOrId("organizations");
                const newOrganization = new Record(orgCollection, {
                    name: providerClaim.get("name"),
                    description: providerClaim.get("description"),
                    logo,
                });
                app.save(newOrganization);
                organizationId = newOrganization.id;

                const orgAuthorization = new Record(
                    app.findCollectionByNameOrId("orgAuthorizations"),
                    {
                        organization: organizationId,
                        user: ownerId,
                        role: utils.getRoleByName("owner")?.id,
                    }
                );
                app.save(orgAuthorization);

                auditLogger(e).info(
                    "created_organization_after_successful_claim",
                    "organizationId",
                    organizationId,
                    "organizationName",
                    providerClaim.get("name"),
                    "ownerId",
                    ownerId,
                    "ownerName",
                    app.findRecordById("users", ownerId).getString("name")
                );
            }

            /**
             * Update the provider
             */

            const provider = app.findRecordById(
                "services",
                providerClaim.get("provider")
            );

            [
                "name",
                "description",
                "country",
                "legal_entity",
                "external_links",
                "external_website_url",
                "documentation_url",
                "contact_email",
            ].forEach((field) => {
                provider.set(field, providerClaim.get(field));
            });

            provider.set("logo", logo);
            provider.set("owner", organizationId);

            app.save(provider);

            auditLogger(e).info(
                "updated_provider_after_successful_claim",
                "providerId",
                provider.id,
                "providerName",
                provider.get("name"),
                "organizationId",
                organizationId,
                "organizationName",
                providerClaim.get("name"),
                "ownerId",
                ownerId
            );

            const user = app.findRecordById("users", ownerId);

            const email = utils.renderEmail("provider-claim-accepted", {
                ProviderName: provider.get("name"),
                DashboardLink: utils.getAppUrl() + "/my",
                AppLogo: utils.getAppLogoUrl(),
                UserName: user.getString("name"),
                AppName: utils.getAppName(),
            });

            utils.sendEmail({
                to: utils.getUserEmailAddressData(user),
                ...email,
            });

            //

            app.delete(providerClaim);
        });
    }
}, "provider_claims");

//

onRecordUpdateRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);
    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    if (utils.isAdminContext(e)) e.next();

    const provider = e.record;
    const originalProvider = provider?.original();
    provider?.set("owner", originalProvider?.get("owner"));

    const agent = utils.getUserFromContext(e);

    auditLogger(e).info(
        "updated_provider",
        "providerId",
        provider?.id,
        "providerName",
        provider?.get("name"),
        "agentId",
        agent?.id,
        "agentName",
        agent?.getString("name")
    );

    e.next();
}, "services");

//

onRecordDeleteRequest((e) => {
    /** @type {Utils} */
    const utils = require(`${__hooks}/utils.js`);
    /** @type {AuditLogger} */
    const auditLogger = require(`${__hooks}/auditLogger.js`);

    e.next();

    auditLogger(e).info("deleted_provider", "providerId", e.record?.id);

    if (
        utils.isAdminContext(e) &&
        e.record?.getString("status") === "in_review"
    ) {
        const provider = e.record;
        const ownerId = provider.getString("owner");
        const user = $app.findRecordById("users", ownerId);

        const email = utils.renderEmail("provider-claim-declined", {
            ProviderName: provider.get("name"),
            UserName: user.getString("name"),
            AppLogo: utils.getAppLogoUrl(),
            AppName: utils.getAppName(),
        });

        utils.sendEmail({
            to: utils.getUserEmailAddressData(user),
            ...email,
        });
    }
}, "services");
