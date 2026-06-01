// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type InteropStatus = 'stable' | 'flaky' | 'failing' | 'broken';

export type InteropMatrixTier = 'group' | 'leaf';

export type InteropMatrixEntity = {
	id: string;
	name: string;
	subtitle?: string;
	avatar_url?: string;
	path: string;
	version_label?: string;
};

export type InteropMatrixGroup = {
	id: string;
	name: string;
	path: string;
	child_count: number;
	avatar_url?: string;
	subtitle?: string;
};

export type InteropMatrixLeaf = InteropMatrixEntity & {
	parent_id?: string;
};

export type InteropAxis = {
	hub_collection: string;
	path_based: boolean;
	tiered: boolean;
};

export type InteropMatrixCell = {
	row_id: string;
	column_id: string;
	row_tier: InteropMatrixTier;
	column_tier: InteropMatrixTier;
	pipeline_count: number;
	total_runs: number;
	total_successes: number;
	success_rate: number;
	status: InteropStatus;
};

export type InteropMatrixResponse = {
	row: InteropAxis;
	column: InteropAxis;
	row_groups: InteropMatrixGroup[];
	row_leaves: InteropMatrixLeaf[];
	column_groups: InteropMatrixGroup[];
	column_leaves: InteropMatrixLeaf[];
	cells: InteropMatrixCell[];
};
