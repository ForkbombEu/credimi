// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { StatusCoordinator, type ProbeReason } from './status-coordinator';
import { checkOnlineStatus, type MobileRunnerReference } from './utils';

//

const onlineByPath = $state<Record<string, boolean>>({});

const coordinator = new StatusCoordinator(checkOnlineStatus, {
	onClear: () => {
		for (const path of Object.keys(onlineByPath)) {
			delete onlineByPath[path];
		}
	},
	onUpdate: (path, online) => {
		onlineByPath[path] = online;
	}
});

export function read(): Readonly<Record<string, boolean>> {
	return onlineByPath;
}

export function isOnline(path: string): boolean | undefined {
	return onlineByPath[path];
}

export function probe(runners: MobileRunnerReference[], options: { reason: ProbeReason }) {
	coordinator.probe(runners, options);
}

export function startPolling(getRunners: () => MobileRunnerReference[]) {
	coordinator.startPolling(getRunners);
}

export function stopPolling() {
	coordinator.stopPolling();
}
