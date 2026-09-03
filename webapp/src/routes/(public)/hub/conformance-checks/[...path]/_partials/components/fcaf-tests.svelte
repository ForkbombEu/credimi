<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { FCAFGroupedTests } from '$lib/pipeline-form/steps/fcaf-validation/grouping.js';

	import { ChevronRightIcon } from '@lucide/svelte';
	import { groupAllTests } from '$lib/pipeline-form/steps/fcaf-validation/grouping.js';

	import { Input } from '@/components/ui/input';
	import { m } from '@/i18n';

	//

	let search = $state('');
	let openGroups = $state<Record<string, boolean>>({});

	const searching = $derived(search.trim() !== '');

	const groups: FCAFGroupedTests[] = $derived.by(() => {
		const query = search.trim().toLowerCase();
		const all = groupAllTests();
		if (!query) return all;
		return all
			.map((group) => ({
				...group,
				tests: group.tests.filter(
					(test) =>
						test.id.toLowerCase().includes(query) ||
						test.title.toLowerCase().includes(query) ||
						test.section.toLowerCase().includes(query)
				),
				groups: group.groups
					.map((subgroup) => ({
						...subgroup,
						tests: subgroup.tests.filter(
							(test) =>
								test.id.toLowerCase().includes(query) ||
								test.title.toLowerCase().includes(query) ||
								test.section.toLowerCase().includes(query)
						)
					}))
					.filter((subgroup) => subgroup.tests.length > 0)
			}))
			.filter((group) => group.tests.length > 0);
	});

	const totalTests = $derived(groups.reduce((sum, group) => sum + group.tests.length, 0));

	function isOpen(key: string): boolean {
		return searching || (openGroups[key] ?? false);
	}

	function toggleOpen(key: string) {
		openGroups[key] = !(openGroups[key] ?? false);
	}
</script>

<div class="rounded-md border">
	<div class="border-b p-2">
		<Input bind:value={search} placeholder={m.Search()} />
	</div>
	<div class="flex items-center justify-between gap-2 border-b px-3 py-2">
		<span class="text-xs text-muted-foreground">{totalTests} {m.Tests()}</span>
	</div>
	<div class="max-h-96 space-y-1 overflow-y-auto p-1">
		{#each groups as group (group.key)}
			<div class="rounded">
				<button
					type="button"
					class="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left hover:bg-muted/50"
					onclick={() => toggleOpen(group.key)}
				>
					<ChevronRightIcon
						class="size-4 shrink-0 text-muted-foreground transition-transform {isOpen(
							group.key
						)
							? 'rotate-90'
							: ''}"
					/>
					<span
						class="inline-block size-2 shrink-0 rounded-full {group.color.bar}"
						aria-hidden="true"
					></span>
					<span class="truncate text-sm font-medium {group.color.text}">
						{group.label}
					</span>
					<span class="ml-auto shrink-0 text-xs text-muted-foreground">
						{group.tests.length}
					</span>
				</button>
				{#if isOpen(group.key)}
					<div class="space-y-1 pb-1 pl-9">
						{#each group.groups as subgroup (subgroup.key)}
							<div>
								<p class="text-[10px] font-medium text-muted-foreground">
									{subgroup.label}
								</p>
								<div class="space-y-0.5 pl-2">
									{#each subgroup.tests as test (test.id)}
										<div class="rounded px-2 py-1">
											<div
												class="truncate font-mono text-xs {group.color
													.text}"
												title={test.id}
											>
												{test.id}
											</div>
											{#if test.title}
												<div
													class="line-clamp-2 text-xs text-muted-foreground"
												>
													{test.title}
												</div>
											{/if}
										</div>
									{/each}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/each}
	</div>
</div>
