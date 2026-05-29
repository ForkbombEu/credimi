// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Standard } from '$lib/conformance/types';

import { m } from '@/i18n';

import { interopEntityData } from './interop-entity-data';
import type { InteropHubCollection } from './interop-hub-collections';
import type { InteropAxis, InteropMatrixCell, InteropMatrixEntity, InteropMatrixResponse } from './types';

import { resolveConformanceCheck } from './resolve-conformance';

export type ViewMatrixEntity = {
	id: string;
	name: string;
	displaySubtitle?: string;
	avatar_url?: string;
	href: string;
};

export type ViewMatrix = {
	cornerLabel: string;
	rows: ViewMatrixEntity[];
	columns: ViewMatrixEntity[];
	cells: Map<string, InteropMatrixCell>;
};

export type ToViewMatrixOptions = {
	standards: readonly Standard[];
};

function hubLabel(hub: string, plural: boolean): string {
	if (!(hub in interopEntityData)) return hub;
	const data = interopEntityData[hub as InteropHubCollection];
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
		displaySubtitle: subtitleOrVersion(enriched.subtitle, enriched.version_label),
		avatar_url: enriched.avatar_url,
		href: hubHref(axis, enriched.path)
	};
}

export function toViewMatrix(
	response: InteropMatrixResponse,
	{ standards }: ToViewMatrixOptions
): ViewMatrix {
	const rowLabel = hubLabel(response.row.hub_collection, false);
	const columnLabel = hubLabel(response.column.hub_collection, false);

	const cells = new Map(
		response.cells.map((cell) => [`${cell.row_id}:${cell.column_id}`, cell] as const)
	);

	return {
		cornerLabel: m.interop_matrix_corner_label({ row: rowLabel, column: columnLabel }),
		rows: response.rows.map((row) => toViewEntity(row, response.row, standards)),
		columns: response.columns.map((column) => toViewEntity(column, response.column, standards)),
		cells
	};
}
