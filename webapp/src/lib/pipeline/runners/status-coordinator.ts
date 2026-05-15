// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';

import type { MobileRunnersResponse } from '@/pocketbase/types';

//

export type ProbeReason = 'periodic' | 'modal' | 'visible';

export const POLL_INTERVAL_MS = 30_000;

type CheckOnlineStatus = (runner: MobileRunnersResponse) => Promise<boolean>;

type StatusListener = {
	onUpdate: (path: string, online: boolean) => void;
	onClear: () => void;
};

export class StatusCoordinator {
	#onlineByPath: Record<string, boolean> = {};
	#seqByPath: Record<string, number> = {};
	#modalGeneration = 0;
	#pollTimer: ReturnType<typeof setInterval> | undefined;
	#pollingActive = false;
	#getRunners: (() => MobileRunnersResponse[]) | undefined;

	constructor(
		private readonly checkOnlineStatus: CheckOnlineStatus,
		private readonly listener: StatusListener
	) {}

	read(): Readonly<Record<string, boolean>> {
		return this.#onlineByPath;
	}

	isOnline(path: string): boolean | undefined {
		return this.#onlineByPath[path];
	}

	probe(runners: MobileRunnersResponse[], options: { reason: ProbeReason }) {
		const uniqueRunners = dedupeRunnersByPath(runners);
		let modalGen = this.#modalGeneration;

		if (options.reason === 'modal') {
			this.#modalGeneration += 1;
			modalGen = this.#modalGeneration;
		}

		for (const runner of uniqueRunners) {
			const path = getPath(runner);
			const seq = (this.#seqByPath[path] ?? 0) + 1;
			this.#seqByPath[path] = seq;

			void this.checkOnlineStatus(runner).then((online) => {
				if (this.#seqByPath[path] !== seq) return;
				if (options.reason === 'modal' && modalGen !== this.#modalGeneration) return;
				this.#onlineByPath[path] = online;
				this.listener.onUpdate(path, online);
			});
		}
	}

	startPolling(getRunners: () => MobileRunnersResponse[]) {
		if (this.#pollingActive) return;

		this.#pollingActive = true;
		this.#getRunners = getRunners;
		this.probe(getRunners(), { reason: 'periodic' });

		this.#pollTimer = setInterval(() => {
			if (!this.#getRunners) return;
			this.probe(this.#getRunners(), { reason: 'periodic' });
		}, POLL_INTERVAL_MS);
	}

	stopPolling() {
		if (this.#pollTimer) clearInterval(this.#pollTimer);
		this.#pollTimer = undefined;
		this.#pollingActive = false;
		this.#getRunners = undefined;
		this.#onlineByPath = {};
		this.listener.onClear();
	}
}

function dedupeRunnersByPath(runners: MobileRunnersResponse[]): MobileRunnersResponse[] {
	const seen = new Set<string>();

	return runners.filter((runner) => {
		const path = getPath(runner);
		if (seen.has(path)) return false;
		seen.add(path);
		return true;
	});
}
