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

/**
 * THROWAWAY: magic id for `/dev/live-view-sheet` preview.
 * Delete with that route folder when preview is done.
 */
export const LIVE_VIEW_PREVIEW_WORKFLOW_ID = '__live_view_preview__';

/** Requests a short-lived live-view URL for one device from the Credimi API. */
export async function requestLiveView(
	workflowId: string,
	runId: string,
	deviceId: string
): Promise<Result<LiveViewStream[], string>> {
	// THROWAWAY: DEV-only fake stream for the Sheet preview page.
	if (import.meta.env.DEV && workflowId === LIVE_VIEW_PREVIEW_WORKFLOW_ID) {
		await new Promise((r) => setTimeout(r, 500));
		return ok([
			{
				device_id: deviceId,
				device_name: deviceId,
				url: `${window.location.origin}/dev/live-view-sheet/fake-stream`
			}
		]);
	}

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
