// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Standard } from '$lib/conformance/types';

import { m } from '@/i18n';

import type {
	InteropAxis,
	InteropMatrixCell,
	InteropMatrixEntity,
	InteropMatrixGroup,
	InteropMatrixLeaf,
	InteropMatrixResponse,
	InteropMatrixTier
} from './types';

import { interopHubEntity, isInteropHubCollection } from './interop-hub-collections';
import { resolveConformanceCheck } from './resolve-conformance';

export type ViewMatrixEntity = {
	id: string;
	name: string;
	displaySubtitle?: string;
	avatar_url?: string;
	href: string;
};

export type ViewMatrixAxisEntry = ViewMatrixEntity & {
	tier: InteropMatrixTier;
	child_count?: number;
};

export type ViewMatrix = {
	cornerLabel: string;
	rows: ViewMatrixAxisEntry[];
	columns: ViewMatrixAxisEntry[];
	cells: Map<string, InteropMatrixCell>;
};

export type BuildVisibleMatrixOptions = {
	standards: readonly Standard[];
	expandedRowGroups: ReadonlySet<string>;
	expandedColumnGroups: ReadonlySet<string>;
};

export function cellKey(
	rowTier: InteropMatrixTier,
	rowId: string,
	colTier: InteropMatrixTier,
	colId: string
): string {
	return `${rowTier}:${rowId}:${colTier}:${colId}`;
}

function hubLabel(hub: string, plural: boolean): string {
	if (!isInteropHubCollection(hub)) return hub;
	const data = interopHubEntity(hub);
	return plural ? (data.labels.plural ?? data.labels.singular) : data.labels.singular;
}

export function hubHref(axis: InteropAxis, path: string): string {
	return `/hub/${axis.hub_collection}/${path}`;
}

export function subtitleOrVersion(
	subtitle: string | null | undefined,
	versionLabel: string | null | undefined
): string | undefined {
	return subtitle ? subtitle : (versionLabel ?? undefined);
}

function displaySubtitleForEntity(
	entity: InteropMatrixEntity,
	axis: InteropAxis
): string | undefined {
	if (axis.hub_collection === 'wallets' && entity.version_label === null) {
		return m.interop_matrix_version_undefined();
	}
	return subtitleOrVersion(entity.subtitle, entity.version_label);
}

function enrichEntityForAxis(
	entity: InteropMatrixEntity,
	axis: InteropAxis,
	standards: readonly Standard[]
): InteropMatrixEntity {
	if (!axis.path_based) return entity;

	const resolved = resolveConformanceCheck(entity.id, standards);
	if (!resolved) return entity;

	return {
		...entity,
		name: resolved.name,
		subtitle: resolved.subtitle ?? undefined,
		avatar_url: resolved.avatar_url ?? undefined
	};
}

function toViewEntity(
	entity: InteropMatrixEntity,
	axis: InteropAxis,
	standards: readonly Standard[]
): ViewMatrixEntity {
	const enriched = enrichEntityForAxis(entity, axis, standards);

	return {
		id: enriched.id,
		name: enriched.name,
		displaySubtitle: displaySubtitleForEntity(enriched, axis),
		avatar_url: enriched.avatar_url,
		href: hubHref(axis, enriched.path)
	};
}

type VisibleAxisItem =
	| { tier: 'group'; group: InteropMatrixGroup }
	| { tier: 'leaf'; leaf: InteropMatrixLeaf };

function visibleAxisItems(
	axis: InteropAxis,
	groups: InteropMatrixGroup[],
	leaves: InteropMatrixLeaf[],
	expandedGroups: ReadonlySet<string>
): VisibleAxisItem[] {
	if (!axis.tiered) {
		return leaves.map((leaf) => ({ tier: 'leaf', leaf }));
	}

	const items: VisibleAxisItem[] = [];
	for (const group of groups) {
		if (expandedGroups.has(group.id)) {
			for (const leaf of leaves) {
				if (leaf.parent_id === group.id) {
					items.push({ tier: 'leaf', leaf });
				}
			}
		} else {
			items.push({ tier: 'group', group });
		}
	}
	return items;
}

function toViewAxisEntry(
	item: VisibleAxisItem,
	axis: InteropAxis,
	standards: readonly Standard[]
): ViewMatrixAxisEntry {
	if (item.tier === 'group') {
		const view = toViewEntity(item.group, axis, standards);
		return {
			...view,
			tier: 'group',
			child_count: item.group.child_count
		};
	}

	return {
		...toViewEntity(item.leaf, axis, standards),
		tier: 'leaf'
	};
}

export function buildVisibleMatrix(
	response: InteropMatrixResponse,
	{ standards, expandedRowGroups, expandedColumnGroups }: BuildVisibleMatrixOptions
): ViewMatrix {
	const rowLabel = hubLabel(response.row.hub_collection, false);
	const columnLabel = hubLabel(response.column.hub_collection, false);

	const cells = new Map(
		response.cells.map(
			(cell) =>
				[
					cellKey(cell.row_tier, cell.row_id, cell.column_tier, cell.column_id),
					cell
				] as const
		)
	);

	const rowItems = visibleAxisItems(
		response.row,
		response.row_groups,
		response.row_leaves,
		expandedRowGroups
	);
	const columnItems = visibleAxisItems(
		response.column,
		response.column_groups,
		response.column_leaves,
		expandedColumnGroups
	);

	return {
		cornerLabel: m.interop_matrix_corner_label({ row: rowLabel, column: columnLabel }),
		rows: rowItems.map((item) => toViewAxisEntry(item, response.row, standards)),
		columns: columnItems.map((item) => toViewAxisEntry(item, response.column, standards)),
		cells
	};
}

/** @deprecated Use {@link buildVisibleMatrix} — kept as alias for callers migrating from Task 8. */
export function toViewMatrix(
	response: InteropMatrixResponse,
	options: BuildVisibleMatrixOptions
): ViewMatrix {
	return buildVisibleMatrix(response, options);
}
