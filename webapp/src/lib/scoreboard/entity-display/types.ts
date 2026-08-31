// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';

//

export type AvatarData = {
	src?: string;
	fallback: string;
	alt: string;
};

export type ChildLink = {
	label: string;
	href: string;
	avatar?: AvatarData;
};

export type Item = {
	key: string;
	name: string;
	href: string;
	avatar?: AvatarData;
	kind?: EntityData;
	caption?: string;
	children?: ChildLink[];
};

export type Layout = 'avatar-only' | 'links-only' | 'compact' | 'full';

export type Align = 'start' | 'end';

export type PocketbaseEntity = {
	id: string;
	collectionName: string;
	name?: string;
	logo?: string;
	logo_url?: string;
	__canonified_path__?: string;
};
