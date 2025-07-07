// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Snippet } from 'svelte';

export type FieldOptions = {
	label: string;
	description: string;
	labelRight?: Snippet;
};
