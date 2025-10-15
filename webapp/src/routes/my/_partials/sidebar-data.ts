// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { appSections } from '$lib/app-state';
import { workflowStatuses } from '$lib/temporal';
import { GlobeIcon, LockIcon, UserIcon } from 'lucide-svelte';

import { m } from '@/i18n';

import type { SidebarGroup } from './sidebar';

//

export const data: SidebarGroup[] = [
	{
		title: m.Services_and_products(),
		items: Object.values(appSections).map((section) => {
			return {
				title: section.label,
				url: `/my/${section.id}`,
				icon: section.icon
			};
		})
	},
	{
		title: m.Test_runs(),
		items: workflowStatuses
			.filter((status) => status !== null)
			.map((status) => ({
				title: status,
				url: `/my/tests/runs?status=${status}`
			}))
	},
	{
		title: m.Settings(),
		items: [
			{
				title: m.Profile(),
				url: '/my/profile',
				icon: UserIcon
			},
			{
				title: m.Organization(),
				url: '/my/organization',
				icon: GlobeIcon
			},
			{
				title: m.Webauthn(),
				url: '/my/profile/webauthn',
				icon: LockIcon
			},
			{
				title: m.API_Keys(),
				url: '/my/profile/api-keys',
				icon: LockIcon
			}
		]
	}
];
