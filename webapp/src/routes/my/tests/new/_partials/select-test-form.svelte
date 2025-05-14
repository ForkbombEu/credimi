<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as RadioGroup from '@/components/ui/radio-group/index.js';
	import { Label } from '@/components/ui/label/index.js';
	import T from '@/components/ui-custom/t.svelte';
	import { watch } from 'runed';
	import { Checkbox as Check } from 'bits-ui';
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import Button from '@/components/ui/button/button.svelte';
	import { ArrowRight } from 'lucide-svelte';
	import type { StandardsWithTestSuites } from './standards-response-schema';
	import { m } from '@/i18n';
	import type { CustomChecksResponse } from '@/pocketbase/types';

	//

	type Props = {
		standards: StandardsWithTestSuites;
		customChecks?: CustomChecksResponse[];
		onSelectTests?: (data: {
			standardId: string;
			suites: string[];
			tests: string[];
			customChecks: string[];
		}) => void;
	};

	let { standards, customChecks = [], onSelectTests }: Props = $props();

	//

	// Makes a flat list `[StandardA:VersionA, StandardA:VersionB, ...]`
	const standardsWithVersions = $derived.by(() => {
		return Object.values(standards).flatMap((standard) =>
			standard.versions.map((version) => ({
				id: `${standard.uid}/${version.uid}`,
				label: `${standard.name} – ${version.name}`,
				description: `${standard.description} (${version.name})`,
				suites: version.suites,
				disabled: standard.disabled
			}))
		);
	});

	// svelte-ignore state_referenced_locally
	let selectedStandardId = $state(
		// Getting the first non disabled standard
		standardsWithVersions.find((s) => !s.disabled)?.id
	);

	const availableTestSuites = $derived.by(() => {
		return standardsWithVersions.find((s) => s.id === selectedStandardId)?.suites;
	});

	let selectedSuites = $state<string[]>([]);
	let selectedTests = $state<string[]>([]);

	watch(
		() => selectedStandardId,
		() => {
			selectedSuites = [];
			selectedTests = [];
			selectedCustomChecks = [];
		}
	);

	//

	const availableCustomChecks = $derived.by(() => {
		return customChecks.filter((check) => check.standard_and_version === selectedStandardId);
	});

	let selectedCustomChecks = $state<string[]>([]);

	//

	// TODO - Reimplement count of selected tests
	// const totalTests = $derived(
	// 	availableTestSuites.reduce((prev, curr) => prev + curr.tests.length, 0)
	// );
</script>

<div class=" flex w-full items-start gap-8 p-8">
	<div class="space-y-4">
		<T tag="h4">{m.Available_standards()}</T>

		<RadioGroup.Root bind:value={selectedStandardId} class="!gap-0">
			{#each standardsWithVersions as option}
				{@const selected = selectedStandardId === option.id}
				{@const disabled = option.disabled}

				<Label
					class={[
						'w-[400px] space-y-1 border-b-2 p-4',
						{
							'border-b-primary bg-secondary ': selected,
							'hover:bg-secondary/35 cursor-pointer border-b-transparent':
								!selected && !disabled,
							'cursor-not-allowed border-b-transparent opacity-50': disabled
						}
					]}
				>
					<div class="flex items-center gap-2">
						<RadioGroup.Item value={option.id} id={option.id} {disabled} />
						<span class="text-lg font-bold">{option.label}</span>
					</div>
					<p class="text-muted-foreground text-sm">{option.description}</p>
				</Label>
			{/each}
		</RadioGroup.Root>
	</div>

	<div class="min-w-0 space-y-8">
		<div class="space-y-4">
			<T tag="h4">Test suites:</T>

			{#if availableTestSuites}
				<Check.Group
					bind:value={selectedTests}
					name="test-suites"
					class="flex flex-col gap-2 space-y-4 overflow-auto"
				>
					{#each availableTestSuites as testSuite}
						{@const testSuiteId = testSuite.uid}
						{@const hasIndividualTests = testSuite.files.length > 0}

						{#snippet suiteLabel()}
							<div>
								<T class="text-md font-bold">{testSuite.name}</T>
								<T class="text-muted-foreground text-xs">
									{testSuite.description}
								</T>
							</div>
						{/snippet}

						{#if !hasIndividualTests}
							<label class="flex items-center gap-3">
								<div class="w-4">
									<Checkbox value={testSuiteId} />
								</div>
								{@render suiteLabel()}
							</label>
						{:else}
							<div class="space-y-2 pl-7">
								<Check.GroupLabel>
									{@render suiteLabel()}
								</Check.GroupLabel>

								{#each testSuite.files as fileId}
									{@const value = `${testSuiteId}/${fileId}`}
									{@const label = fileId.split('.').slice(0, -1).join('.')}
									<Label class="flex items-center gap-2 font-mono text-xs">
										<Checkbox {value} />
										<span>{label}</span>
									</Label>
								{/each}
							</div>
						{/if}
					{/each}
				</Check.Group>
			{:else}
				<p class="text-muted-foreground text-sm">No test suites available</p>
			{/if}
		</div>

		{#if availableCustomChecks.length > 0}
			<div class="space-y-4">
				<T tag="h4">Custom checks:</T>

				<Check.Group
					bind:value={selectedCustomChecks}
					name="test-suites"
					class="flex flex-col gap-2 overflow-auto"
				>
					{#each availableCustomChecks as check}
						<Label class="flex items-start gap-2 text-sm">
							<Checkbox value={check.id} />
							<div>
								<T>{check.name}</T>
								{#if check.description}
									<T class="text-muted-foreground text-xs">{check.description}</T>
								{/if}
							</div>
						</Label>
					{/each}
				</Check.Group>
			</div>
		{/if}
	</div>
</div>

<div class="bg-background/20 sticky bottom-0 border-t p-4 px-8 backdrop-blur-lg">
	<div class="mx-auto flex w-full max-w-screen-xl items-center justify-between">
		<!-- TODO - Reimplement count of selected tests -->
		<div></div>
		<!-- <p class="text-gray-400">
			<span class="rounded-sm bg-gray-200 p-1 font-bold text-black"
				>{selectedTests.length}</span
			>
			/ {totalTests}
			selected
		</p> -->
		<Button
			disabled={selectedTests.length + selectedCustomChecks.length === 0}
			onclick={() =>
				onSelectTests?.({
					standardId: selectedStandardId!,
					suites: selectedSuites,
					tests: selectedTests,
					customChecks: selectedCustomChecks
				})}
		>
			Next step <ArrowRight />
		</Button>
	</div>
</div>
