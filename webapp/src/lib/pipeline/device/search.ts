// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { DeviceRecord } from './types';

export function filterDevices(devices: readonly DeviceRecord[], text: string): DeviceRecord[] {
	const search = text.trim().toLowerCase();
	if (!search) return [...devices];

	return devices.filter(
		(device) =>
			device.name.toLowerCase().includes(search) || device.path.toLowerCase().includes(search)
	);
}
