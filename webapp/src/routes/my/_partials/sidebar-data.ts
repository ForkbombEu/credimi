// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { appSections, testRunsSection } from '$lib/marketplace/sections';
import { GlobeIcon, HomeIcon, LockIcon, StoreIcon, UserIcon } from 'lucide-svelte';

import { m } from '@/i18n';

import type { SidebarGroup } from './sidebar';

import { WorkflowStatusesSidebarSection } from './statuses-section.svelte';

//

export const data: SidebarGroup[] = [
	{
		items: [
			{
				title: m.Home(),
				url: '/my',
				icon: HomeIcon
			},
			{
				title: m.Marketplace(),
				url: '/marketplace',
				icon: StoreIcon
			}
		]
	},
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
		title: testRunsSection.label,
		items: [
			{
				title: testRunsSection.label,
				url: testRunsSection.id,
				icon: testRunsSection.icon
			},
			WorkflowStatusesSidebarSection
		]
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
