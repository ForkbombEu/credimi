<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Suite } from '$lib/standards';

	import SectionCard from '$lib/layout/section-card.svelte';
	import Footer from '$start-checks-form/_utils/footer.svelte';
	import { ArrowRight, GitBranch, HelpCircle, Home } from '@lucide/svelte';
	import { Checkbox as Check } from 'bits-ui';

	import LinkExternal from '@/components/ui-custom/linkExternal.svelte';
	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Button from '@/components/ui/button/button.svelte';
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import { Label } from '@/components/ui/label/index.js';
	import * as RadioGroup from '@/components/ui/radio-group/index.js';
	import * as Select from '@/components/ui/select/index.js';
	import { m } from '@/i18n';

	import type { SelectChecksForm } from './select-checks-form.svelte.js';

	import SmallErrorDisplay from '../_utils/small-error-display.svelte';
	import OpenidSuiteTable from './openid-suite-files-table.svelte';

	//

	const OPENID_SUITE_UID = 'openid_conformance_suite';

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
				<SectionCard
					title={m.Test_suites()}
					subtitle={m.Select_official_test_suites_subtitle()}
				>
					{#snippet headerActions()}
						<div class="flex gap-2">
							{#if form.selectedStandard?.standard_url}
								<LinkExternal
									href={form.selectedStandard.standard_url}
									text="{form.selectedStandard.name} {m.Standard()}"
									icon={HelpCircle}
									title={m.Learn_about_standard({
										name: form.selectedStandard.name
									})}
								/>
							{/if}

							{#if form.selectedVersion?.specification_url}
								<LinkExternal
									href={form.selectedVersion.specification_url}
									text="{form.selectedVersion.name} {m.Spec()}"
									icon={GitBranch}
									title={m.View_specification({
										name: form.selectedVersion.name
									})}
								/>
							{/if}
						</div>
					{/snippet}

					{#if form.availableSuitesWithoutTests.length > 0}
						{@render SuitesWithoutTestsSelect()}
					{/if}

					{#if form.availableSuitesWithTests.length > 0}
						{@render SuitesWithTestsSelect()}
					{/if}
				</SectionCard>
			{/if}

			{#if form.availableCustomChecks.length > 0}
				<SectionCard title={m.Custom_checks()} subtitle={m.Select_custom_checks_subtitle()}>
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
	<Check.Group
		bind:value={() => form.selectedTests, (v) => (form.selectedTests = v)}
		class="flex flex-col gap-8"
	>
		{#each form.availableSuitesWithTests as suite}
			<div class="space-y-4">
				<Check.GroupLabel>
					{@render suiteLabel(suite)}
				</Check.GroupLabel>

				<div class="w-full space-y-3 overflow-auto">
					{#if suite.uid === OPENID_SUITE_UID}
						<OpenidSuiteTable suiteFiles={suite.files} suiteUid={suite.uid} />
					{:else}
						{#each suite.files as fileId}
							{@const value = `${suite.uid}/${fileId}`}
							{@const label = fileId.split('.').slice(0, -1).join('.')}
							<Label class="flex items-center gap-2  font-mono text-xs">
								<Checkbox {value} />
								<span>{label}</span>
							</Label>
						{/each}
					{/if}
				</div>
			</div>
		{/each}
	</Check.Group>
{/snippet}

{#snippet CustomChecksSelect()}
	<Check.Group
		bind:value={() => form.selectedCustomChecksIds, (v) => (form.selectedCustomChecksIds = v)}
		name="test-suites"
		class="flex flex-col gap-4 overflow-auto"
	>
		{#each form.availableCustomChecks as check}
			<Label class="flex items-start gap-3 text-sm">
				<div class="w-4 pt-0.5">
					<Checkbox value={check.id} />
				</div>
				<div class="min-w-0 flex-1 space-y-2">
					<div>
						<T class="font-medium">{check.name}</T>
						{#if check.description}
							<T class="text-muted-foreground text-xs">
								{check.description}
							</T>
						{/if}
					</div>

					{#if check.homepage || check.repository}
						<div class="flex flex-wrap gap-2 text-xs">
							{#if check.homepage}
								<LinkExternal
									href={check.homepage}
									text={m.Homepage()}
									icon={Home}
									title={m.Custom_check_homepage()}
								/>
							{/if}

							{#if check.repository}
								<LinkExternal
									href={check.repository}
									text={m.Repository()}
									icon={GitBranch}
									title={m.Custom_check_repository()}
								/>
							{/if}
						</div>
					{/if}
				</div>
			</Label>
		{/each}
	</Check.Group>
{/snippet}

{#snippet suiteLabel(suite: Suite)}
	<div class="space-y-3">
		<div>
			<T class="text-md font-bold">{suite.name}</T>
			<T class="text-muted-foreground text-xs">
				{suite.description}
			</T>
		</div>

		{#if suite.help || suite.homepage || suite.repository}
			<div class="flex flex-wrap gap-3 text-xs">
				{#if suite.help}
					<LinkExternal
						href={suite.help}
						text={m.Instructions()}
						icon={HelpCircle}
						title={m.Help_and_instructions()}
					/>
				{/if}

				{#if suite.homepage}
					<LinkExternal
						href={suite.homepage}
						text={m.Homepage()}
						icon={Home}
						title={m.Official_homepage()}
					/>
				{/if}

				{#if suite.repository}
					<LinkExternal
						href={suite.repository}
						text={m.Repository()}
						icon={GitBranch}
						title={m.Source_code_repository()}
					/>
				{/if}
			</div>
		{/if}
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
