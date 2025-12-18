// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Component } from 'svelte';

//

export interface WithComponentProps<T extends WithComponent = WithComponent> {
	self: T;
}

export interface WithComponent {
	Component: Component<WithComponentProps, Record<string, never>, ''>;
}
