// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Component } from 'svelte';

//

export interface WithComponent {
	Component: Component<{ self: WithComponent }, Record<string, never>, ''>;
}
