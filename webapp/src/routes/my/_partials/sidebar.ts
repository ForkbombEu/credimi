// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Component, Snippet } from 'svelte';

import type { IconComponent } from '@/components/types';

//

export type SidebarItem = {
	title: string;
	url: string;
	notification?: boolean;
	icon?: IconComponent;
	children?: Omit<SidebarItem, 'children'>[];
	component?: Component<SidebarItemComponentProps>;
};

export type SidebarGroup = {
	title?: string;
	items: (SidebarItem | Snippet | (() => ReturnType<Snippet>))[];
};

export type SidebarItemComponentProps = { title: string };
