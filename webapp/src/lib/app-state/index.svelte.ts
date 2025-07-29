// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { OrganizationsResponse } from '@/pocketbase/types';

export const userOrganization = $state<{ current: OrganizationsResponse | undefined }>({
	current: undefined
});
