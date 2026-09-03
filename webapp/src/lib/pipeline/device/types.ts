// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type DeviceRecord = {
	name: string;
	path: string;
	runnerId?: string;
	runnerName?: string;
	description?: string;
	isOwned: boolean;
	isPublished: boolean;
	isOnline: boolean;
	url?: string;
	type?: string;
	queueLength?: number;
};
