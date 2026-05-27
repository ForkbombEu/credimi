// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type InteropStatus = 'stable' | 'flaky' | 'failing' | 'broken';

export type InteropMatrixEntity = {
	id: string;
	name: string;
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

export type InteropMatrixResponse = {
	mode: string;
	row_axis: string;
	column_axis: string;
	rows: InteropMatrixEntity[];
	columns: InteropMatrixEntity[];
	cells: InteropMatrixCell[];
};
