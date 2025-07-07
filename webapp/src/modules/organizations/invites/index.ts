// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createSessionStorageHandlers } from '@/utils/storage';

type OrganizationInviteSession = {
	organizationId: string;
	inviteId: string;
	email: string;
	userId?: string | undefined;
};

export const OrganizationInviteSession =
	createSessionStorageHandlers<OrganizationInviteSession>('organization_invite');
