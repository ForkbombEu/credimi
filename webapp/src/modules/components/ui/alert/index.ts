// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import Description from './alert-description.svelte';
import Title from './alert-title.svelte';
import Root from './alert.svelte';
export { alertVariants, type AlertVariant } from './alert.svelte';

export {
	Root,
	Description,
	Title,
	//
	Root as Alert,
	Description as AlertDescription,
	Title as AlertTitle
};
