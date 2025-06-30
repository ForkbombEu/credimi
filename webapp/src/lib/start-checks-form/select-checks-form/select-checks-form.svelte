<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Suite } from '$lib/standards';
	import type { SelectChecksForm } from './select-checks-form.svelte.js';
	import * as RadioGroup from '@/components/ui/radio-group/index.js';
	import * as Select from '@/components/ui/select/index.js';
	import { Label } from '@/components/ui/label/index.js';
	import T from '@/components/ui-custom/t.svelte';
	import { Checkbox as Check } from 'bits-ui';
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import Button from '@/components/ui/button/button.svelte';
	import { ArrowRight } from 'lucide-svelte';
	import { m } from '@/i18n';
	import SectionCard from '$lib/layout/section-card.svelte';
	import Footer from '$start-checks-form/_utils/footer.svelte';
	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import SmallErrorDisplay from '../_utils/small-error-display.svelte';

	//

	type Props = {
		form: SelectChecksForm;
	};

	const { form }: Props = $props();
</script>

<div class="flex flex-col items-start gap-4 md:flex-row">
	<SectionCard title={m.Available_standards()} class="w-full !space-y-2 md:w-auto">
		{@render StandardSelect()}
	</SectionCard>

	<div class="w-full space-y-4 md:min-w-0 md:grow">
		{#if form.selectedStandard}
			<SectionCard title={m.Standard_version()}>
				{@render VersionSelect()}
			</SectionCard>
		{/if}

		{#if form.selectedVersion}
			{#if form.availableSuites.length > 0}
				<SectionCard title={m.Test_suites()}>
					{#if form.availableSuitesWithoutTests.length > 0}
						{@render SuitesWithoutTestsSelect()}
					{/if}

					{#if form.availableSuitesWithTests.length > 0}
						{@render SuitesWithTestsSelect()}
					{/if}
				</SectionCard>
			{/if}

			{#if form.availableCustomChecks.length > 0}
				<SectionCard title={m.Custom_checks()}>
					{@render CustomChecksSelect()}
				</SectionCard>
			{/if}
		{/if}
	</div>
</div>

{@render FormFooter()}

<!-- Snippets -->

{#snippet StandardSelect()}
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
	<!-- This binding style is needed to avoid a svelte warning -->
	<Check.Group
		bind:value={() => form.selectedTests, (v) => (form.selectedTests = v)}
		class="flex flex-col gap-6 overflow-auto"
	>
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
	<!-- This binding style is needed to avoid a svelte warning -->
	<Check.Group
		bind:value={() => form.selectedTests, (v) => (form.selectedTests = v)}
		class="flex flex-col gap-6"
	>
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
							<span>{label}</span>
						</Label>
					{/each}
				</div>
			</div>
		{/each}
	</Check.Group>
{/snippet}

{#snippet CustomChecksSelect()}
	<!-- This binding style is needed to avoid a svelte warning -->
	<Check.Group
		bind:value={() => form.selectedCustomChecksIds, (v) => (form.selectedCustomChecksIds = v)}
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
	<Footer>
		{#snippet left()}
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
					{#if form.selectedCustomChecksIds.length > 0}
						{@render CountItem(form.selectedCustomChecksIds.length, m.Custom_checks())}
					{/if}
				</ul>
			{/if}
		{/snippet}

		{#snippet right()}
			{#if form.loadingError}
				<SmallErrorDisplay error={form.loadingError} />
			{/if}

			<Button
				disabled={!form.isValid}
				onclick={() => {
					form.submit();
				}}
			>
				{m.Next_step()}
				<ArrowRight />
			</Button>
		{/snippet}
	</Footer>
{/snippet}

{#if form.isLoading}
	<LoadingDialog />
{/if}
