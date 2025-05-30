<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import { ChecksConfigsFormComponent } from './checks-configs-form/utils.js';
	import { SelectTestsFormComponent } from './select-tests-form/index.js';
	import { StartChecksForm, type StartChecksFormProps } from './start-checks-form.svelte.js';
	import * as Tabs from '@/components/ui/tabs/index.js';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { m } from '@/i18n/index.js';

	//

	const props: StartChecksFormProps = $props();
	const form = new StartChecksForm(props);

	//

	type FormState = typeof form.state;

	const tabs: { id: FormState; label: string }[] = [
		{ id: 'select-tests', label: '1. Standard and test suite' },
		{ id: 'fill-values', label: '2. Key values and JSONs' }
	];

	$effect(() => {
		if (form.state === 'fill-values') {
			scrollTo({ top: 0, behavior: 'instant' });
		}
	});

	// const customCheckId = $derived(page.url.searchParams.get(queryParams.customCheckId));

	// $effect(() => {
	// 	if (!customCheckId) return;
	// 	const customCheck = data.customChecks.find((check) => check.id === customCheckId);
	// 	if (!customCheck) return;

	// 	compositeTestId = customCheck.standard_and_version;
	// 	selectedCustomChecksIds = [customCheckId];
	// 	formState = 'fill-values';
	// });
</script>

<div class="space-y-12 pb-0 pt-8">
	<Tabs.Root value={form.state} class="w-full">
		<Tabs.List class="bg-secondary flex">
			<Tabs.Trigger
				value={tabs[0].id}
				class="data-[state=inactive]:hover:bg-primary/10 grow data-[state=inactive]:text-black"
				onclick={() => {
					form.backToSelectTests();
				}}
			>
				{tabs[0].label}
			</Tabs.Trigger>
			<Tabs.Trigger value={tabs[1].id} class="grow" disabled={form.state === 'select-tests'}>
				{tabs[1].label}
			</Tabs.Trigger>
		</Tabs.List>
	</Tabs.Root>
</div>

<div class="bg-background relative w-full rounded-t-md shadow-sm">
	{#if form.state === 'select-tests'}
		<SelectTestsFormComponent form={form.selectTestsForm}>
			{#snippet footerRight()}
				{#if form.loadingError}
					<div
						class="rounded-md border border-red-500 bg-red-100 px-1 py-0.5 text-xs text-red-700"
					>
						<p class="space-x-1">
							<span class="font-bold">{m.Error()}:</span>
							<span>{form.loadingError.message}</span>
						</p>
					</div>
				{/if}
			{/snippet}
		</SelectTestsFormComponent>
	{:else if form.state === 'fill-values' && form.checksConfigsFormProps}
		<ChecksConfigsFormComponent {...form.checksConfigsFormProps} />
	{/if}
</div>

{#if form.isLoadingData}
	<LoadingDialog />
{/if}

<!-- <hr />
{#if customChecks.length > 0}

<div class="space-y-4">
	<T tag="h2" class="text-lg font-bold">Review custom checks</T>

	{#each customChecks as check}
		{@const logo = pb.files.getURL(check, check.logo)}
		<div class="flex items-start gap-4 rounded-md border p-2 px-4">
			<Avatar src={logo} class="rounded-sm" fallback={check.name.slice(0, 2)} />
			<div>
				<T class="font-bold">{check.name}</T>
				<T class="mb-2 font-mono text-xs">{check.standard_and_version}</T>
				{#if check.description}
					<T class="mb-2 text-sm text-gray-400">{check.description}</T>
				{/if}
				<pre class="rounded-sm bg-black p-3 text-xs text-white">{check.yaml}</pre>
				<JsonSchemaForm
					schema={check.input_json_schema as object}
					options={{ hideTitle: true, hideSubmitButton: true }}
					onUpdate={(data) => {
						console.log(data);
					}}
				/>
			</div>
		</div>
	{/each}
</div>
{/if} -->

<!-- 

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
						{#if customChecks.length > 0}
							<span class="text-muted-foreground">and</span>
							{customChecks.length} custom checks
						{/if}
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
</div> -->
