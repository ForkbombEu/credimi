// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';

import { getPath } from '$lib/utils';

import type { Item, PocketbaseEntity } from './types';

//

export function getPocketbaseEntityHref(entity: PocketbaseEntity): string {
	return `/hub/${entity.collectionName}/${getPath(entity)}`;
}

export function fromPocketbaseEntity(entity: PocketbaseEntity, kind?: EntityData): Item {
	const name = entity.name ?? '';
	return {
		key: entity.id,
		name,
		href: getPocketbaseEntityHref(entity),
		avatar: {
			src: 'logo_url' in entity ? entity.logo_url : undefined,
			fallback: name.slice(0, 2),
			alt: name
		},
		kind
	};
}

export function fromPocketbaseEntities(entities: PocketbaseEntity[], kind?: EntityData): Item[] {
	return entities.map((entity) => fromPocketbaseEntity(entity, kind));
}
