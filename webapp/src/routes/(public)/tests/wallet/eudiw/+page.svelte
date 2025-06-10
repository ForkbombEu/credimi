<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { createForm } from '@/forms';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';
	import Separator from '@/components/ui/separator/separator.svelte';
	import { pb } from '@/pocketbase/index.js';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { MediaQuery } from 'svelte/reactivity';
	import { m } from '@/i18n';
	import Step from '../_partials/step.svelte';
	import QrLink from '../_partials/qr-link.svelte';
	import SuccessForm from '../_partials/success-form.svelte';
	import FailureForm from '../_partials/failure-form.svelte';

	//

	let { data } = $props();
	const { qr, workflowId, namespace } = $derived(data);

	let pageStatus = $state<'fresh' | 'success' | 'already_answered'>('fresh');

	const successForm = createForm({
		adapter: zod(z.object({})),
		onSubmit: async () => {
			await pb.send('/api/compliance/confirm-success', {
				method: 'POST',
				body: {
					workflow_id: workflowId,
					namespace: namespace
				}
			});
			pageStatus = 'success';
		},
		onError: ({ setFormError, errorMessage }) => {
			handleErrorMessage(errorMessage, setFormError);
		}
	});

	const failureForm = createForm({
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
			pageStatus = 'success';
		},
		onError: ({ setFormError, errorMessage }) => {
			handleErrorMessage(errorMessage, setFormError);
		}
	});

	function handleErrorMessage(message: string, errorFallback: () => void) {
		const lowercased = message.toLowerCase();
		if (lowercased.includes('signal') && lowercased.includes('failed'))
			pageStatus = 'already_answered';
		else errorFallback();
	}

	//

	const sm = new MediaQuery('min-width: 640px');
</script>

<PageContent>
	<T tag="h1" class="mb-4">Wallet test</T>
	<div class="space-y-4">
		{#if qr}
			<Step n="1" text="Scan this QR with the wallet app to start the check">
				<div
					class="bg-primary/10 ml-16 mt-4 flex flex-col items-center justify-center rounded-md p-2 sm:flex-row"
				>
					<QrLink {qr} />
				</div>
			</Step>
		{:else}
			<Alert variant="destructive">
				<T class="font-bold">{m.Error_check_failed()}</T>
				<T>
					{m.An_error_happened_during_the_check_please_read_the_logs_for_more_information()}
				</T>
			</Alert>
		{/if}

		<Step n="2" text="Confirm the result">
			<div class="ml-16 flex flex-col gap-8 sm:flex-row">
				{#if pageStatus == 'fresh'}
					{#if data.qr}
						<SuccessForm {successForm} />
						<Separator orientation={sm.current ? 'vertical' : 'horizontal'} />
					{/if}
					<FailureForm {failureForm} />
				{:else if pageStatus == 'success'}
					<Alert variant="info">Your response was submitted! Thanks :)</Alert>
				{:else if pageStatus == 'already_answered'}
					<Alert variant="info">This test was already confirmed</Alert>
				{/if}
			</div>
		</Step>
	</div>
</PageContent>
