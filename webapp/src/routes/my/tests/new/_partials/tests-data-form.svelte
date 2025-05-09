<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import FieldConfigFormShared from './field-config-form-shared.svelte';
	import FieldConfigForm from './field-config-form.svelte';
	import { createTestListInputSchema, type FieldsResponse } from './logic';
	import { createForm, Form, SubmitButton, FormError } from '@/forms';
	import { zod } from 'sveltekit-superforms/adapters';
	import { Store } from 'runed';
	import * as Popover from '@/components/ui/popover';
	import Button from '@/components/ui/button/button.svelte';
	import { pb } from '@/pocketbase';
	import { goto } from '$app/navigation';
	import type { CustomChecksResponse } from '@/pocketbase/types';
	//

	type Props = {
		data?: FieldsResponse | undefined;
		testId: string;
		customChecks: CustomChecksResponse[];
	};

	let { data = { normalized_fields: [], specific_fields: {} }, testId = 'openid4vp' }: Props =
		$props();

	//

	let sharedData = $state<Record<string, unknown>>({});

	const defaultFieldsIds = Object.values(data.normalized_fields).map((f) => f.CredimiID);

	//

	const form = createForm({
		adapter: zod(createTestListInputSchema(data)),
		onSubmit: async ({ form }) => {
			await pb.send(`/api/compliance/${testId}/save-variables-and-start`, {
				method: 'POST',
				body: form.data
			});
			await goto(`/my/tests/runs`);
		},
		options: {
			resetForm: false
		}
	});

	const { form: formData, validateForm } = form;
	const formState = new Store(formData);

	const testsIds = $derived(Object.keys(data.specific_fields));

	const incompleteTestsIdsPromise = $derived.by(() => {
		formState.current;
		return validateForm().then((result) => testsIds.filter((test) => test in result.errors));
	});

	const completeTestsCount = $derived(
		incompleteTestsIdsPromise.then((tests) => testsIds.length - tests.length)
	);

	const completionStatusPromise = $derived(
		Promise.all([completeTestsCount, incompleteTestsIdsPromise])
	);

	//

	const SHARED_FIELDS_ID = 'shared-fields';
</script>

<div class="mx-auto w-full max-w-screen-xl space-y-16 p-8">
	{#if data.normalized_fields.length > 0}
		<div class="space-y-4">
			<h2 id={SHARED_FIELDS_ID} class="text-lg font-bold">Shared fields</h2>
			<FieldConfigFormShared
				fields={data.normalized_fields}
				onUpdate={(form) => (sharedData = form)}
			/>
		</div>

		<hr />
	{/if}

	{#each Object.entries(data.specific_fields) as [testId, testData], index}
		<div class="space-y-4">
			<h2 id={testId} class="text-lg font-bold">
				{testId}
			</h2>
			<FieldConfigForm
				fields={testData.fields}
				jsonConfig={JSON.parse(testData.content)}
				defaultValues={sharedData}
				{defaultFieldsIds}
				onValidUpdate={(form) => {
					$formData[testId] = form;
				}}
				onInvalidUpdate={() => {
					// @ts-expect-error
					$formData[testId] = undefined;
				}}
			/>
		</div>

		{#if index < Object.keys(data.specific_fields).length - 1}
			<hr />
		{/if}
	{/each}
</div>

<div class="bg-background/80 sticky bottom-0 border-t py-4 backdrop-blur-lg">
	<Form
		{form}
		hide={['submit_button', 'error']}
		class="mx-auto w-full max-w-screen-xl space-y-4 px-8"
	>
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				{#await completionStatusPromise then [completeTestsCount, incompleteTestsIds]}
					<p>
						{completeTestsCount}/{testsIds.length} configs complete
					</p>
					{#if incompleteTestsIds.length}
						<Popover.Root>
							<Popover.Trigger
								class="rounded-md p-1 hover:cursor-pointer hover:bg-gray-200"
							>
								{#snippet child({ props })}
									<Button {...props} variant="outline" class="px-3">
										View incomplete configs ({incompleteTestsIds.length})
									</Button>
								{/snippet}
							</Popover.Trigger>
							<Popover.Content class="dark w-fit">
								<ul class="space-y-1 text-sm">
									{#each incompleteTestsIds as testId}
										<li>
											<a
												class="underline hover:no-underline"
												href={`#${testId}`}
											>
												{testId}
											</a>
										</li>
									{/each}
								</ul>
							</Popover.Content>
						</Popover.Root>
					{:else}
						<p>✅</p>
					{/if}
				{/await}
			</div>

			<SubmitButton>Next</SubmitButton>
		</div>

		<FormError />
	</Form>
</div>
