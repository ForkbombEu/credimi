// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SuperForm } from 'sveltekit-superforms';

import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod/v3';

import type { GenericRecord } from '@/utils/types';

import { createForm } from '@/forms/form';
import { pb } from '@/pocketbase';

//

export type FeedbackFormProps = {
	workflowId: string;
	namespace: string;
};

export class FeedbackForms {
	status = $state<'fresh' | 'success' | 'already_answered'>('fresh');

	successForm: SuperForm<GenericRecord>;
	failureForm: SuperForm<{ reason: string }>;

	constructor({ workflowId, namespace }: FeedbackFormProps) {
		this.successForm = createForm({
			adapter: zod(z.object({})),
			onSubmit: async () => {
				await pb.send('/api/compliance/confirm-success', {
					method: 'POST',
					body: {
						workflow_id: workflowId,
						namespace: namespace
					}
				});
				this.status = 'success';
			},
			onError: ({ setFormError, errorMessage }) => {
				this.handleErrorMessage(errorMessage, () => setFormError(errorMessage));
			}
		});

		this.failureForm = createForm({
			adapter: zod(z.object({ reason: z.string().min(3) })),
			onSubmit: async ({
				form: {
					data: { reason }
				}
			}) => {
				await pb.send('/api/compliance/notify-failure', {
					method: 'POST',
					body: {
						workflow_id: workflowId,
						namespace: namespace,
						reason: reason
					}
				});
				this.status = 'success';
			},
			onError: ({ setFormError, errorMessage }) => {
				this.handleErrorMessage(errorMessage, () => setFormError(errorMessage));
			}
		});
	}

	private handleErrorMessage(message: string, errorFallback: () => void) {
		const lowercased = message.toLowerCase();
		if (lowercased.includes('signal') && lowercased.includes('failed'))
			this.status = 'already_answered';
		else errorFallback();
	}
}
