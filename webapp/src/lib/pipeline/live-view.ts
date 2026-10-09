// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { err, ok, Result } from 'true-myth/result';

import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors';

//

export type LiveViewStream = { device_id: string; device_name: string; url: string };

type LiveViewResponse = { streams: LiveViewStream[] };

/** Display-ready device row from pipeline execution summaries / workflow describe. */
export type ExecutionDevice = {
	device_id: string;
	name: string;
	live_view: boolean;
};

/** Requests a short-lived live-view URL for one device from the Credimi API. */
export async function requestLiveView(
	workflowId: string,
	runId: string,
	deviceId: string
): Promise<Result<LiveViewStream[], string>> {
	try {
		const res = await pb.send<LiveViewResponse>('/api/pipeline/live-view', {
			method: 'POST',
			body: { workflow_id: workflowId, run_id: runId, device_id: deviceId },
			requestKey: null
		});
		return ok(res.streams ?? []);
	} catch (e) {
		return err(getExceptionMessage(e));
	}
}

//

export type MaestroCommandStatus =
	| 'RUNNING'
	| 'COMPLETED'
	| 'FAILED'
	| 'WARNED'
	| 'SKIPPED'
	| 'PENDING';

/** Latest state of one Maestro command row, as served by the runner. */
export type MaestroCommand = {
	seq: number;
	flow_id: string;
	index: number;
	parent: number;
	depth: number;
	description: string;
	status: MaestroCommandStatus;
};

export type MaestroCommandsPage = { commands: MaestroCommand[]; next: number };

export type MaestroFlowGroup = { flowId: string; commands: MaestroCommand[] };

/** Runner long-poll URL returning command rows with `seq > after`. */
export function liveViewCommandsUrl(streamUrl: string, after: number): string {
	return `${streamUrl.replace(/\/+$/, '')}/commands?after=${after}`;
}

/**
 * Plain cross-origin GET (no credentials, no custom headers) so the runner can
 * answer without a CORS preflight. err(status) on HTTP error, err(0) on network
 * error; an abort is rethrown.
 */
export async function fetchLiveViewCommands(
	streamUrl: string,
	after: number,
	signal: AbortSignal
): Promise<Result<MaestroCommandsPage, number>> {
	try {
		const res = await fetch(liveViewCommandsUrl(streamUrl, after), {
			signal,
			cache: 'no-store'
		});
		if (!res.ok) return err(res.status);
		return ok((await res.json()) as MaestroCommandsPage);
	} catch (e) {
		if (e instanceof DOMException && e.name === 'AbortError') throw e;
		return err(0);
	}
}

/** Replaces rows with the same (flow_id, index) in place and appends unseen rows in incoming order. */
export function mergeMaestroCommands(
	current: MaestroCommand[],
	incoming: MaestroCommand[]
): MaestroCommand[] {
	const key = (c: MaestroCommand) => `${c.flow_id}\u0000${c.index}`;
	const merged = [...current];
	const positions = new Map(merged.map((c, i) => [key(c), i]));
	for (const command of incoming) {
		const position = positions.get(key(command));
		if (position === undefined) {
			positions.set(key(command), merged.length);
			merged.push(command);
		} else {
			merged[position] = command;
		}
	}
	return merged;
}

/** Groups consecutive rows by flow_id, preserving order. */
export function groupMaestroCommands(commands: MaestroCommand[]): MaestroFlowGroup[] {
	const groups: MaestroFlowGroup[] = [];
	for (const command of commands) {
		const last = groups.at(-1);
		if (last?.flowId === command.flow_id) last.commands.push(command);
		else groups.push({ flowId: command.flow_id, commands: [command] });
	}
	return groups;
}
