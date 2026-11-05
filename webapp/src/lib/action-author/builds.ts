// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { DeviceRecord } from '$lib/pipeline/device/types';

import type { OrganizationsResponse, WalletsResponse } from '@/pocketbase/types';

import { m } from '@/i18n';

import type { AuthorDevice, BuildOption } from './api';

//

/** The `organization/wallet` path that prefixes the wallet's build identifiers. */
export function walletPath(
	organization: Pick<OrganizationsResponse, 'canonified_name'>,
	wallet: Pick<WalletsResponse, 'canonified_name'>
): string {
	return `${organization.canonified_name}/${wallet.canonified_name}`;
}

/** The wallet's builds: its store build (`path`) and its versions (`path/…`), in API order. */
export function walletBuilds(builds: BuildOption[], path: string): BuildOption[] {
	return builds.filter((b) => b.identifier === path || b.identifier.startsWith(path + '/'));
}

export function buildPlatforms(builds: BuildOption[]): Set<string> {
	return new Set(builds.flatMap((b) => b.platforms));
}

export function buildsForPlatform(builds: BuildOption[], platform: string): BuildOption[] {
	return builds.filter((b) => b.platforms.includes(platform));
}

/** The devices of the builds' platforms, as device-select list items. */
export function toDeviceRecords(devices: AuthorDevice[], platforms: Set<string>): DeviceRecord[] {
	return devices
		.filter((d) => platforms.has(d.platform))
		.map((d) => ({
			name: d.name,
			path: d.path,
			runnerName: d.runner_name,
			isOnline: d.is_online,
			queueLength: d.queue_length,
			type: d.type,
			isOwned: false,
			isPublished: true,
			description: `${d.platform === 'ios' ? 'iOS simulator' : 'Android'} · ${m.Runner_of({ owner: d.runner_owner })} · ${d.model}`,
			unavailableReason: d.agent_available ? undefined : m.Action_author_agent_not_running()
		}));
}

export function buildLabel(build: BuildOption): string {
	return build.source === 'store' ? m.Installed_from_the_store() : build.version_tag;
}
