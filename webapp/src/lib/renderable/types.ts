// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Component } from 'svelte';

//

export type SelfProps<T> = { self: T };

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface Renderable<T = any> {
	Component: Component<SelfProps<T>>;
}
