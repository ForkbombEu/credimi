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
