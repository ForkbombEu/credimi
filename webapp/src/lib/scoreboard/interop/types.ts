// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { InteropMode } from './modes';

export type { InteropMode } from './modes';

export type InteropStatus = 'stable' | 'flaky' | 'failing' | 'broken';

export type InteropMatrixEntity = {
	id: string;
	name: string;
	subtitle?: string;
	avatar_url?: string;
	path: string;
	version_label?: string;
};

export type InteropMatrixCell = {
	row_id: string;
	column_id: string;
	pipeline_count: number;
	total_runs: number;
	total_successes: number;
	success_rate: number;
	status: InteropStatus;
};

export type InteropAxis = {
	key: string;
	hub_collection: string;
	path_based: boolean;
};

export type InteropMatrixResponse = {
	mode: InteropMode;
	row: InteropAxis;
	column: InteropAxis;
	rows: InteropMatrixEntity[];
	columns: InteropMatrixEntity[];
	cells: InteropMatrixCell[];
};
