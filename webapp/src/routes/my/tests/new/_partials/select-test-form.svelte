<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { StandardWithTestSuites } from './logic';
	import * as RadioGroup from '@/components/ui/radio-group/index.js';
	import { Label } from '@/components/ui/label/index.js';
	import T from '@/components/ui-custom/t.svelte';
	import { watch } from 'runed';
	import { Checkbox as Check } from 'bits-ui';
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import Button from '@/components/ui/button/button.svelte';
	import { ArrowRight } from 'lucide-svelte';

	//

	type Props = {
		standards: StandardWithTestSuites[];
		onSelectTests?: (standardId: string, tests: string[]) => void;
	};

	let { standards, onSelectTests }: Props = $props();

	let selectedStandardId = $state(standards[0].id);

	let compositeTestId = $derived.by(
		() =>
			`${selectedStandardId}/${standards.find((s) => s.id === selectedStandardId)?.testSuites[0].id}`
	);

	const availableTestSuites = $derived(
		standards.find((s) => s.id === selectedStandardId)?.testSuites ?? []
	);
	const totalTests = $derived(
		availableTestSuites.reduce((prev, curr) => prev + curr.tests.length, 0)
	);

	let selectedTests = $state<string[]>([]);

	watch(
		() => selectedStandardId,
		() => {
			selectedTests = [];
		}
	);
</script>

<div class="mx-auto flex w-full max-w-screen-xl items-start gap-8 p-8">
	<div class="space-y-4">
		<T tag="h4">Available standards:</T>

		<RadioGroup.Root bind:value={selectedStandardId} class="!gap-0">
			{#each standards as test}
				{@const selected = selectedStandardId === test.id}
				{@const disabled = test.testSuites.length === 0 || test.disabled}

				<Label
					class={[
						'w-[300px] space-y-1 border-b-2 p-4',
						{
							'border-b-primary bg-secondary ': selected,
							'hover:bg-secondary/35 cursor-pointer border-b-transparent':
								!selected && !disabled,
							'cursor-not-allowed border-b-transparent opacity-50': disabled
						}
					]}
				>
					<div class="flex items-center gap-2">
						<RadioGroup.Item value={test.id} id={test.id} {disabled} />
						<span class="text-lg font-bold">{test.label}</span>
					</div>
					<p class="text-muted-foreground text-sm">{test.description}</p>
				</Label>
			{/each}
		</RadioGroup.Root>
	</div>

	<div class="min-w-0 space-y-4">
		<T tag="h4">Test suites:</T>

		<Check.Group
			bind:value={selectedTests}
			name="test-suites"
			class="flex flex-col gap-2 overflow-auto"
		>
			{#each availableTestSuites as testSuite}
				<div class="space-y-2">
					<Check.GroupLabel class="text-sm text-gray-400 underline underline-offset-4">
						{testSuite.label}
					</Check.GroupLabel>
					{#each testSuite.tests as testId}
						<Label class="flex items-center gap-2 font-mono text-xs">
							<Checkbox name="test-suites" value={testId} />
							<span>{testId.replace('.json', '')}</span>
						</Label>
					{/each}
				</div>
			{/each}
		</Check.Group>
	</div>
</div>

<div class="bg-background/20 sticky bottom-0 border-t p-4 px-8 backdrop-blur-lg">
	<div class="mx-auto flex w-full max-w-screen-xl items-center justify-between">
		<p class="text-gray-400">
			<span class="rounded-sm bg-gray-200 p-1 font-bold text-black">
				{selectedTests.length}
			</span>
			/ {totalTests}
			selected
		</p>
		<Button
			disabled={selectedTests.length === 0}
			onclick={() => onSelectTests?.(compositeTestId, selectedTests)}
		>
			Next step <ArrowRight />
		</Button>
	</div>
</div>
