<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import TextareaField from '@/forms/fields/textareaField.svelte';
	import { Form, SubmitButton, createForm } from '@/forms';
	import { QrCode } from '@/qr';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';
	import Separator from '@/components/ui/separator/separator.svelte';
	import { pb } from '@/pocketbase/index.js';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Label } from '@/components/ui/label';
	import { MediaQuery } from 'svelte/reactivity';
	import WorkflowLogs from '@/components/ui-custom/workflowLogs.svelte';
	import ParamsChecker from './_partials/params-checker.svelte';

	//

	let { data } = $props();

	let pageStatus = $state<'fresh' | 'success' | 'already_answered'>('fresh');

	const successForm = createForm({
		adapter: zod(z.object({})),
		onSubmit: async () => {
			await pb.send('/api/compliance/confirm-success', {
				method: 'POST',
				body: {
					workflow_id: data.workflowId
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
					workflow_id: data.workflowId,
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
	<ParamsChecker>
		{#snippet ifValid({ qr, workflowId })}
			<div class="space-y-4">
				<div class="step-container">
					{@render Step(1, 'Scan this QR with the wallet app to start the check')}

					<div
						class="bg-primary/10 ml-16 mt-4 flex flex-col items-center justify-center rounded-md p-2 sm:flex-row"
					>
						<QrCode src={qr} class="size-40 rounded-sm" />

						<p
							class="text-primary max-w-sm break-all p-4 font-mono text-xs hover:underline"
						>
							{qr}
						</p>
					</div>
				</div>

				<div class="step-container">
					{@render Step(2, 'Follow the procedure on the wallet app')}
					<div class="ml-16">
						<WorkflowLogs {workflowId} />
					</div>
				</div>

				<div class="step-container">
					{@render Step(3, 'Confirm the result')}

					<div class="ml-16 flex flex-col gap-8 sm:flex-row">
						{#if pageStatus == 'fresh'}
							<div class="grow basis-1">
								<Form form={successForm}>
									{#snippet submitButton()}
										<div class="space-y-2">
											<Label for="success">If the test succeeded:</Label>
											<SubmitButton
												id="success"
												class="w-full bg-green-600 hover:bg-green-700"
											>
												Confirm test success
											</SubmitButton>
										</div>
									{/snippet}
								</Form>
							</div>

							<Separator orientation={sm.current ? 'vertical' : 'horizontal'} />

							<div class="grow basis-1">
								<Form form={failureForm} hideRequiredIndicator class="space-y-2">
									<TextareaField
										form={failureForm}
										name="reason"
										options={{
											label: 'If something went wrong, please tell us what:'
										}}
									/>
									{#snippet submitButton()}
										<SubmitButton class="w-full bg-red-600 hover:bg-red-700">
											Notify issue
										</SubmitButton>
									{/snippet}
								</Form>
							</div>
						{:else if pageStatus == 'success'}
							<Alert variant="info">Your response was submitted! Thanks :)</Alert>
						{:else if pageStatus == 'already_answered'}
							<Alert variant="info">This test was already confirmed</Alert>
						{/if}
					</div>
				</div>
			</div>
		{/snippet}
	</ParamsChecker>
</PageContent>

{#snippet Step(n: number, text: string)}
	<div class="flex items-center gap-4">
		<div
			class="bg-primary text-primary-foreground flex size-12 shrink-0 items-center justify-center rounded-full text-lg font-semibold"
		>
			<p>{n}</p>
		</div>
		<T class="text-primary font-semibold">{text}</T>
	</div>
{/snippet}

<style lang="postcss">
	.step-container {
		@apply bg-secondary rounded-xl p-4;
	}
</style>
