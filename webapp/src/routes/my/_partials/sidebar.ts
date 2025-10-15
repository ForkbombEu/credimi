// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { IconComponent } from '@/components/types';

export type SidebarItem = {
	title: string;
	url: string;
	isActive?: boolean;
	notification?: boolean;
	icon?: IconComponent;
};

export type SidebarGroup = {
	title: string;
	items: SidebarItem[];
};
