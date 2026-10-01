// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	EXTERNAL_VERSION,
	GLOBAL_DEVICE,
	type SelectedDevice,
	type SelectedVersion
} from '$pipeline-form/execution-target/types.js';

import { m } from '@/i18n/index.js';

export function getVersionLabel(version: SelectedVersion) {
	return version === EXTERNAL_VERSION ? m.Installed_from_external_source() : `v. ${version.tag}`;
}

export function getDeviceLabel(device: SelectedDevice) {
	return device === GLOBAL_DEVICE ? m.Choose_later() : device.name;
}
