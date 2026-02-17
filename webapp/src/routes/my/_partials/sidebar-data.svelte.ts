// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	GlobeIcon,
	HomeIcon,
	HourglassIcon,
	LockIcon,
	SheetIcon,
	StoreIcon,
	UserIcon
} from '@lucide/svelte';
import { page } from '$app/state';
import { Pipeline } from '$lib';
import { baseSections, entities } from '$lib/global';
import { WORKFLOW_STATUS_QUERY_PARAM } from '$lib/workflows';
import { SvelteURL } from 'svelte/reactivity';

import { m } from '@/i18n';

import type { SidebarGroup, SidebarItem } from './sidebar';

import { ALL_WORKFLOW_STATUSES } from '../tests/runs/_partials';
import { IDS } from '../wallets/utils';
import WorkflowItem from './components/workflow-item.svelte';

//

export function getSidebarData(): SidebarGroup[] {
	return data;
}

const data: SidebarGroup[] = $derived([
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
		items: baseSections.map((section) => {
			let children: SidebarItem[] | undefined;
			if (section.slug === 'wallets') {
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
				title: section.labels.plural ?? '',
				url: `/my/${section.slug}`,
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
				url: `/my/${entities.test_runs.slug}`,
				icon: entities.test_runs.icon,
				children: ALL_WORKFLOW_STATUSES.filter((status) => status !== null).map(
					(status) => {
						const base = '/my/tests/runs';
						const url = new SvelteURL(base, page.url.origin);
						if (page.url.pathname.includes(base)) {
							page.url.searchParams.forEach((value, key) => {
								const excludeKeys = [
									Pipeline.Workflows.LIMIT_PARAM,
									Pipeline.Workflows.OFFSET_PARAM
								];
								if (excludeKeys.includes(key)) return;
								url.searchParams.set(key, value);
							});
						}
						url.searchParams.set(WORKFLOW_STATUS_QUERY_PARAM, status);
						return {
							title: status,
							url: url.toString(),
							component: WorkflowItem
						};
					}
				)
			},
			{
				title: m.Scheduled_pipelines(),
				url: `/my/pipelines/schedule`,
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
]);
