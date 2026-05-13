// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';
import { getPath } from '$lib/utils';

import { pb } from '@/pocketbase';

import type { Item, PocketbaseEntity } from './types';

//

export function getPocketbaseEntityHref(entity: PocketbaseEntity): string {
	return `/marketplace/${entity.collectionName}/${getPath(entity)}`;
}

export function fromPocketbaseEntity(entity: PocketbaseEntity, kind?: EntityData): Item {
	return {
		key: entity.id,
		name: entity.name,
		href: getPocketbaseEntityHref(entity),
		avatar:
			'logo' in entity && entity.logo
				? {
						src: pb.files.getURL(entity, entity.logo),
						fallback: entity.name.slice(0, 2),
						alt: entity.name
					}
				: undefined,
		kind
	};
}

export function fromPocketbaseEntities(
	entities: PocketbaseEntity[],
	kind?: EntityData
): Item[] {
	return entities.map((entity) => fromPocketbaseEntity(entity, kind));
}
