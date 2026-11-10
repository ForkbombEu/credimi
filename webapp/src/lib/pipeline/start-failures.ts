// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { toast } from 'svelte-sonner';
import { z } from 'zod/v3';

import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase/index.js';

/**
 * Realtime topic on which the server announces queued or scheduled pipeline runs of an
 * organization that could not start (`pipelinealerts.StartFailureTopic` in Go). The failure
 * happens when the devices free up, possibly hours after the run was queued, so the enqueue
 * response cannot carry it. Nothing is stored: only open tabs receive it.
 */
export function startFailureTopic(organizationId: string) {
	return `pipeline_start_failures/${organizationId}`;
}

const startFailureSchema = z.object({
	pipeline_name: z.string(),
	pipeline_identifier: z.string(),
	ticket_id: z.string(),
	error: z.string()
});

export type StartFailure = z.infer<typeof startFailureSchema>;

export function parseStartFailure(data: unknown): StartFailure | undefined {
	const result = startFailureSchema.safeParse(data);
	return result.success ? result.data : undefined;
}

/** Shows an error toast for every run of the organization that fails to start. Resolves to the unsubscribe function. */
export function subscribeToStartFailures(organizationId: string) {
	return pb.realtime.subscribe(startFailureTopic(organizationId), (data) => {
		const failure = parseStartFailure(data);
		if (!failure) return;
		toast.error(m.Pipeline_failed_to_start({ name: failure.pipeline_name }), {
			description: failure.error,
			duration: Number.POSITIVE_INFINITY,
			action: {
				label: m.View(),
				onClick: () => goto(`/my/pipelines/${failure.pipeline_identifier}`)
			}
		});
	});
}
