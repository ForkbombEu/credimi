// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

type EmailData = {
    "confirm-email": [
        "AppName",
        "UserName",
        "AppName",
        "VerificationLink",
        "VerificationLink",
        "VerificationLink"
    ],
    "join-organization": [
        "OrganizationName",
        "AppName",
        "OrganizationName",
        "UserName",
        "Editor",
        "OrganizationName",
        "DashboardLink"
    ],
    "member-removal": [
        "OrganizationName",
        "AppName",
        "UserName",
        "OrganizationName",
        "AppName",
        "DashboardLink"
    ],
    "membership-request-accepted": [
        "OrganizationName",
        "AppName",
        "UserName",
        "OrganizationName",
        "AppName",
        "OrganizationName",
        "DashboardLink"
    ],
    "membership-request-declined": [
        "AppName",
        "UserName",
        "OrganizationName",
        "AppName",
        "DashboardLink"
    ],
    "membership-request-new": [
        "UserName",
        "OrganizationName",
        "AppName",
        "Admin",
        "UserName",
        "OrganizationName",
        "AppName",
        "DashboardLink"
    ],
    "membership-request-pending": [
        "OrganizationName",
        "AppName",
        "Admin",
        "PendingNumber",
        "DashboardLink"
    ],
    "new-organization": [
        "OrganizationName",
        "AppName",
        "AppName",
        "UserName",
        "OrganizationName",
        "AppName",
        "DashboardLink"
    ],
    "provider-claim-accepted": [
        "ProviderName",
        "AppName",
        "ProviderName",
        "UserName",
        "ProviderName",
        "AppName",
        "ProviderName",
        "DashboardLink"
    ],
    "provider-claim-declined": [
        "ProviderName",
        "AppName",
        "ProviderName",
        "UserName",
        "ProviderName"
    ],
    "reset-password": [
        "AppName",
        "username",
        "AppName",
        "resetLink"
    ],
    "role-change": [
        "OrganizationName",
        "AppName",
        "Admin",
        "OrganizationName",
        "UserName",
        "Admin",
        "OrganizationName",
        "Membership",
        "DashboardLink"
    ],
    "user-invitation": [
        "Editor",
        "OrganizationName",
        "AppName",
        "OrganizationName",
        "Editor",
        "OrganizationName",
        "AppName",
        "InvitationLink",
        "InvitationLink",
        "AppName"
    ],
    "user-welcome": [
        "AppName",
        "AppName",
        "AppName",
        "UserName",
        "AppName",
        "DashboardLink"
    ]
}