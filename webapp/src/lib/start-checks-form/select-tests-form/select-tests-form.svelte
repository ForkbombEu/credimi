<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Suite } from '$lib/standards';
	import { SelectTestsForm } from './select-tests-form.svelte.js';

	import * as RadioGroup from '@/components/ui/radio-group/index.js';
	import * as Select from '@/components/ui/select/index.js';
	import { Label } from '@/components/ui/label/index.js';
	import T from '@/components/ui-custom/t.svelte';
	import { Checkbox as Check } from 'bits-ui';
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import Button from '@/components/ui/button/button.svelte';
	import { ArrowRight } from 'lucide-svelte';
	import { m } from '@/i18n';

	//

	type Props = {
		form: SelectTestsForm;
	};

	const { form }: Props = $props();
</script>

<div class="flex flex-col items-start gap-8 p-8 md:flex-row">
	<div class="w-full space-y-4">
		{@render StandardSelect()}
	</div>

	<div class="w-full space-y-8 md:min-w-0 md:grow">
		{#if form.selectedStandard}
			<div class="space-y-4">
				{@render VersionSelect()}
			</div>
		{/if}

		{#if form.selectedVersion}
			{#if form.availableSuites.length > 0}
				<div class="space-y-4">
					<T tag="h4">{m.Test_suites()}</T>

					{#if form.availableSuitesWithoutTests.length > 0}
						{@render SuitesWithoutTestsSelect()}
					{/if}

					{#if form.availableSuitesWithTests.length > 0}
						{@render SuitesWithTestsSelect()}
					{/if}
				</div>
			{/if}

			{#if form.availableCustomChecks.length > 0}
				<div class="space-y-4">
					{@render CustomChecksSelect()}
				</div>
			{/if}
		{/if}
	</div>
</div>

{@render FormFooter()}

<!-- Snippets -->

{#snippet StandardSelect()}
	<T tag="h4">{m.Available_standards()}</T>

	<RadioGroup.Root bind:value={form.selectedStandardId} class="!gap-0" required>
		{#each form.availableStandards as option}
			{@const selected = form.selectedStandardId === option.uid}
			{@const disabled = option.disabled}

			<Label
				class={[
					'w-full space-y-1 border-b-2 p-4 md:w-[400px]',
					{
						'border-b-primary bg-secondary ': selected,
						'hover:bg-secondary/35 cursor-pointer border-b-transparent':
							!selected && !disabled,
						'cursor-not-allowed border-b-transparent opacity-50': disabled
					}
				]}
			>
				<div class="flex items-center gap-2">
					<RadioGroup.Item value={option.uid} id={option.uid} {disabled} />
					<span class="text-lg font-bold">{option.name}</span>
				</div>
				<p class="text-muted-foreground text-sm">{option.description}</p>
			</Label>
		{/each}
	</RadioGroup.Root>
{/snippet}

{#snippet VersionSelect()}
	<T tag="h4">{m.Version()}</T>
	<Select.Root type="single" bind:value={form.selectedVersionId}>
		<Select.Trigger>
			{form.selectedVersion?.name ?? m.Select_a_version()}
		</Select.Trigger>
		<Select.Content>
			{#each form.availableVersions as version (version.uid)}
				<Select.Item value={version.uid} label={version.name}>
					{version.name}
				</Select.Item>
			{/each}
		</Select.Content>
	</Select.Root>
{/snippet}

{#snippet SuitesWithoutTestsSelect()}
	<Check.Group bind:value={form.selectedSuites} class="flex flex-col gap-4 overflow-auto">
		{#each form.availableSuitesWithoutTests as suite}
			<label class="flex items-center gap-3">
				<div class="w-4">
					<Checkbox value={suite.uid} />
				</div>
				{@render suiteLabel(suite)}
			</label>
		{/each}
	</Check.Group>
{/snippet}

{#snippet SuitesWithTestsSelect()}
	<Check.Group bind:value={form.selectedTests} class="flex flex-col gap-4 ">
		{#each form.availableSuitesWithTests as suite}
			<div class="space-y-3 border-l-4 pl-4">
				<Check.GroupLabel>
					{@render suiteLabel(suite)}
				</Check.GroupLabel>

				<div class="w-full space-y-3 overflow-auto">
					{#each suite.files as fileId}
						{@const value = `${suite.uid}/${fileId}`}
						{@const label = fileId.split('.').slice(0, -1).join('.')}
						<Label class="flex items-center gap-2  font-mono text-xs">
							<Checkbox {value} />
							<span>{label + label}</span>
						</Label>
					{/each}
				</div>
			</div>
		{/each}
	</Check.Group>
{/snippet}

{#snippet CustomChecksSelect()}
	<T tag="h4">{m.Custom_checks()}</T>

	<Check.Group
		bind:value={form.selectedCustomChecks}
		name="test-suites"
		class="flex flex-col gap-2 overflow-auto"
	>
		{#each form.availableCustomChecks as check}
			<Label class="flex items-start gap-2 text-sm">
				<Checkbox value={check.id} />
				<div>
					<T>{check.name}</T>
					{#if check.description}
						<T class="text-muted-foreground text-xs">
							{check.description}
						</T>
					{/if}
				</div>
			</Label>
		{/each}
	</Check.Group>
{/snippet}

{#snippet suiteLabel(suite: Suite)}
	<div>
		<T class="text-md font-bold">{suite.name}</T>
		<T class="text-muted-foreground text-xs">
			{suite.description}
		</T>
	</div>
{/snippet}

{#snippet FormFooter()}
	<div class="bg-background/20 sticky bottom-0 border-t p-4 px-8 backdrop-blur-lg">
		<div class="mx-auto flex max-w-screen-xl items-center justify-between">
			<div class="flex text-sm">
				{#if form.hasSelection}
					<p>{m.Current_selection()}:</p>
					<ul class="flex items-center divide-x">
						{#snippet CountItem(count: number, label: string)}
							<li class="px-2">
								<span class="text-primary font-bold">{count}</span>
								{label}
							</li>
						{/snippet}

						{#if form.selectedSuites.length > 0}
							{@render CountItem(form.selectedSuites.length, m.Suites())}
						{/if}
						{#if form.selectedTests.length > 0}
							{@render CountItem(form.selectedTests.length, m.Tests())}
						{/if}
						{#if form.selectedCustomChecks.length > 0}
							{@render CountItem(form.selectedCustomChecks.length, m.Custom_checks())}
						{/if}
					</ul>
				{/if}
			</div>
			<!-- <p class="text-gray-400">
			<span class="rounded-sm bg-gray-200 p-1 font-bold text-black"
				>{selectedTests.length}</span
			>
			/ {totalTests}
			selected
		</p> -->
			<Button
				disabled={!form.isValid}
				onclick={() => {
					form.submit();
				}}
			>
				{m.Next_step()}
				<ArrowRight />
			</Button>
		</div>
	</div>
{/snippet}
