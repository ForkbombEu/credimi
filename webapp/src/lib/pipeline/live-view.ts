// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { VideoIcon } from '@lucide/svelte';
import { runWithLoading } from '$lib/utils';
import { toast } from 'svelte-sonner';
import { err, ok, Result } from 'true-myth/result';

import type { DropdownMenuItem } from '@/components/ui-custom/dropdown-menu.svelte';

import { m } from '@/i18n';
import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors';

import type { ExecutionSummary } from './workflows';

//

export type LiveViewStream = { device_id: string; device_name: string; url: string };

type LiveViewResponse = { streams: LiveViewStream[] };

export type LiveViewDeviceTarget = { deviceId: string; deviceLabel: string | null };

/** Devices of a running execution that can get a "Watch live" action. */
export function liveViewDeviceTargets(w: ExecutionSummary): LiveViewDeviceTarget[] {
	if (w.status !== 'Running' || !w.device_ids?.length) return [];
	const single = w.device_ids.length === 1;
	return w.device_ids.map((deviceId) => ({
		deviceId,
		deviceLabel: single ? null : (deviceId.split('/').pop() ?? deviceId)
	}));
}

export async function requestLiveView(
	workflowId: string,
	runId: string,
	deviceId?: string
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

/**
 * Opens the live view in a new tab. The tab is opened synchronously inside the
 * click so popup blockers allow it, then pointed at the stream once known.
 * Returns the streams when the caller must let the user choose among several.
 */
export async function watchLive(
	workflowId: string,
	runId: string,
	deviceId?: string
): Promise<LiveViewStream[] | undefined> {
	const tab = window.open('about:blank', '_blank');

	const result = await runWithLoading({
		fn: () => requestLiveView(workflowId, runId, deviceId),
		showSuccessToast: false
	});

	if (!result || result.isErr) {
		tab?.close();
		toast.error(
			`${m.Live_view_open_failed()}: ${result?.isErr ? result.error : 'Unexpected error'}`
		);
		return undefined;
	}

	const streams = result.value;
	if (streams.length === 1) {
		if (tab) {
			tab.opener = null;
			tab.location.href = streams[0].url;
		} else {
			window.location.href = streams[0].url;
		}
		return undefined;
	}

	tab?.close();
	return streams;
}

export function liveViewDropdownItems(w: ExecutionSummary): DropdownMenuItem[] {
	return liveViewDeviceTargets(w).map((target) => ({
		label: target.deviceLabel
			? m.Watch_live_on_device({ device: target.deviceLabel })
			: m.Watch_live(),
		icon: VideoIcon,
		onclick: () => watchLive(w.execution.workflowId, w.execution.runId, target.deviceId)
	}));
}
