// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { X } from 'lucide-svelte';
import type { Snippet } from 'svelte';
import type { HTMLAnchorAttributes } from 'svelte/elements';

//

export type IconComponent = typeof X;

export interface Link extends HTMLAnchorAttributes {
	title: string;
}

export interface LinkWithIcon extends Link {
	icon?: IconComponent;
}

export type SnippetFunction<T> = (props: T) => ReturnType<Snippet>;

export interface NavItem {
	href: string;
	label: string;
	icon?: IconComponent;
	/**
	 * Controls where this item appears:
	 * - 'both': appears in both desktop and mobile (default)
	 * - 'desktop-only': only appears in desktop navigation
	 * - 'mobile-only': only appears in mobile menu
	 */
	display?: 'both' | 'desktop-only' | 'mobile-only';
	/**
	 * Custom click handler
	 */
	onClick?: () => void;
	/**
	 * Additional CSS classes
	 */
	class?: string;
}
