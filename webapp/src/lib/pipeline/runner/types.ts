// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type RunnerRecord = {
	name: string;
	path: string;
	description?: string;
	isOwned: boolean;
	isPublished: boolean;
	isOnline: boolean;
	url?: string;
	type?: string;
	queueLength?: number;
};
