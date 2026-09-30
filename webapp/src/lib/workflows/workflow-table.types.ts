// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

type TableColumn =
	| 'type'
	| 'workflow'
	| 'status'
	| 'date'
	| 'start'
	| 'end'
	| 'duration'
	| 'actions';

export interface HideColumnsProp {
	hideColumns?: TableColumn[];
}
