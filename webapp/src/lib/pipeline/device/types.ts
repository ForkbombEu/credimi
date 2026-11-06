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
	/** Whether `isOnline` is current without the device catalog; the catalog decides otherwise. */
	statusKnown?: boolean;
	url?: string;
	type?: string;
	queueLength?: number;
	unavailableReason?: string;
};
