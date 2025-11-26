// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { appSections, testRunsSection } from '$lib/marketplace/sections';
import { workflowStatuses } from '$lib/temporal';
import { WORKFLOW_STATUS_QUERY_PARAM } from '$lib/workflows';
import { GlobeIcon, HomeIcon, LockIcon, StoreIcon, UserIcon } from 'lucide-svelte';

import { m } from '@/i18n';

import type { SidebarGroup, SidebarItem } from './sidebar';

import { IDS } from '../wallets/utils';
import WorkflowItem from './components/workflow-item.svelte';

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
			let children: SidebarItem[] | undefined;
			if (section.id === 'wallets') {
				children = [
					{
						title: m.Your_wallets(),
						url: `/my/wallets#${IDS.YOUR_WALLETS}`
					},
					{
						title: m.Public_wallets(),
						url: `/my/wallets#${IDS.PUBLIC_WALLETS}`
					}
				];
			}
			return {
				title: section.label,
				url: `/my/${section.id}`,
				icon: section.icon,
				children
			};
		})
	},
	{
		title: testRunsSection.label,
		items: [
			{
				title: testRunsSection.label,
				url: testRunsSection.id,
				icon: testRunsSection.icon,
				children: workflowStatuses
					.filter((status) => status !== null)
					.map((status) => ({
						title: status,
						url: `/my/tests/runs?${WORKFLOW_STATUS_QUERY_PARAM}=${status}`,
						component: WorkflowItem
					}))
			}
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
