// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Cog, Home, Network, Users } from '@lucide/svelte';

import type { LinkWithIcon } from '@/components/types';

import { m } from '@/i18n';

import type { OrgRole } from '.';

//

export function createOrganizationLinks(
	organizationId: string,
	userRole: OrgRole = 'member'
): LinkWithIcon[] {
	const base = (path = '') => `/my/organizations/${organizationId}${path}`;

	const links: LinkWithIcon[] = [
		{
			title: m.Home(),
			href: base(),
			icon: Home
		},
		{
			title: m.Services(),
			href: base('/providers'),
			icon: Network
		},
		{
			title: m.Members(),
			href: base('/members'),
			icon: Users
		}
	];

	if (userRole == 'owner') {
		links.push({
			title: m.Settings(),
			href: base('/settings'),
			icon: Cog
		});
	}

	return links;
}
