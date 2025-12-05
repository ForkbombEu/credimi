// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { appSections, testRunsSection } from '$lib/marketplace/sections';
import { workflowStatuses } from '$lib/temporal';
import { WORKFLOW_STATUS_QUERY_PARAM } from '$lib/workflows';
import {
	GlobeIcon,
	HomeIcon,
	HourglassIcon,
	LockIcon,
	SheetIcon,
	StoreIcon,
	UserIcon
} from 'lucide-svelte';

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
				url: '/',
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
		title: m.marketplace_items(),
		items: Object.values(appSections)
			.filter((section) => section.id !== 'conformance-checks')
			.map((section) => {
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
		title: m.workflows(),
		items: [
			{
				title: m.workflow_runs(),
				url: testRunsSection.id,
				icon: testRunsSection.icon,
				children: workflowStatuses
					.filter((status) => status !== null)
					.map((status) => ({
						title: status,
						url: `/my/tests/runs?${WORKFLOW_STATUS_QUERY_PARAM}=${status}`,
						component: WorkflowItem
					}))
			},
			{
				title: m.Scheduled_workflows(),
				url: '/my/tests/runs/scheduled',
				icon: HourglassIcon
			}
		]
	},
	{
		title: m.tools(),
		items: [
			{
				title: m.manual_conformance_checks(),
				url: '/my/tests/new',
				icon: SheetIcon
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
