<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { FCAFTestCatalogEntry } from '$lib/fcaf/tests.generated.js';
	import type { SelfProp } from '$lib/renderable';

	import { ChevronRightIcon } from '@lucide/svelte';
	import { FCAF_SUITE } from '$lib/fcaf/tests.generated.js';
	import { WithLabel } from '$pipeline-form/steps/_partials/index.js';

	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { m } from '@/i18n';

	import type { FCAFValidationStepForm } from './fcaf-validation-step-form.svelte.js';

	import { groupTests } from './grouping.js';

	//

	let { self: form }: SelfProp<FCAFValidationStepForm> = $props();

	let search = $state('');
	let openGroups = $state<Record<string, boolean>>({});

	const selectedIds = $derived(form.selectedTestIds);
	const selectedCount = $derived(selectedIds.length);
	const searching = $derived(search.trim() !== '');

	const filteredGroups = $derived.by(() => {
		const query = search.trim().toLowerCase();
		const filtered = query
			? form.availableTests.filter(
					(test) =>
						test.id.toLowerCase().includes(query) ||
						test.title.toLowerCase().includes(query) ||
						test.section.toLowerCase().includes(query)
				)
			: form.availableTests;

		return groupTests(filtered);
	});

	function isOpen(key: string): boolean {
		return searching || (openGroups[key] ?? false);
	}

	function toggleOpen(key: string) {
		openGroups[key] = !(openGroups[key] ?? false);
	}

	function groupSelectedCount(tests: FCAFTestCatalogEntry[]): number {
		return tests.filter((test) => selectedIds.includes(test.id)).length;
	}

	function groupAllSelected(tests: FCAFTestCatalogEntry[]): boolean {
		return tests.length > 0 && groupSelectedCount(tests) === tests.length;
	}

	function groupPartial(tests: FCAFTestCatalogEntry[]): boolean {
		const count = groupSelectedCount(tests);
		return count > 0 && count < tests.length;
	}

	function toggleGroup(tests: FCAFTestCatalogEntry[]) {
		const ids = tests.map((test) => test.id);
		const selected = selectedIds;
		if (ids.every((id) => selected.includes(id))) {
			form.setTestIds(selected.filter((id) => !ids.includes(id)));
			return;
		}
		const merged = [...selected];
		for (const id of ids) {
			if (!merged.includes(id)) merged.push(id);
		}
		form.setTestIds(merged);
	}
</script>

<div class="space-y-6 p-4">
	<WithLabel label={m.Tests()} required>
		<div class="rounded-md border">
			<div class="border-b p-2">
				<Input bind:value={search} placeholder={m.Search()} />
			</div>
			<div class="flex items-center justify-between gap-2 border-b px-3 py-2">
				<span class="text-xs text-muted-foreground">
					{selectedCount} / {form.availableTests.length}
				</span>
				<div class="flex gap-1">
					<Button variant="ghost" size="sm" onclick={() => form.selectAllTestIds()}>
						{m.Select_all()}
					</Button>
					<Button variant="ghost" size="sm" onclick={() => form.clearTestIds()}>
						{m.Clear_selection()}
					</Button>
				</div>
			</div>
			<div class="max-h-72 space-y-1 overflow-y-auto p-1">
				{#each filteredGroups as group (group.key)}
					<div class="rounded">
						<div class="flex items-center gap-2 rounded px-2 py-1.5 hover:bg-muted/50">
							<button
								type="button"
								class="flex min-w-0 flex-1 items-center gap-2 text-left"
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
									class="inline-block size-2 shrink-0 rounded-full {group.color
										.bar}"
									aria-hidden="true"
								></span>
								<span class="truncate text-sm font-medium {group.color.text}">
									{group.label}
								</span>
							</button>
							<span class="ml-auto shrink-0 text-xs text-muted-foreground">
								{groupSelectedCount(group.tests)}/{group.tests.length}
							</span>
							<input
								type="checkbox"
								class="shrink-0 accent-primary"
								checked={groupAllSelected(group.tests)}
								indeterminate={groupPartial(group.tests)}
								onchange={() => toggleGroup(group.tests)}
							/>
						</div>
						{#if isOpen(group.key)}
							<div class="space-y-1 pb-1 pl-9">
								{#each group.groups as subgroup (subgroup.key)}
									<div class="rounded">
										<div
											class="flex items-center gap-2 rounded px-2 py-1 hover:bg-muted/50"
										>
											<button
												type="button"
												class="flex min-w-0 flex-1 items-center gap-2 text-left"
												onclick={() =>
													toggleOpen('{group.key}/{subgroup.key}')}
											>
												<ChevronRightIcon
													class="size-3.5 shrink-0 text-muted-foreground transition-transform {isOpen(
														'{group.key}/{subgroup.key}'
													)
														? 'rotate-90'
														: ''}"
												/>
												<span class="truncate text-xs font-medium">
													{subgroup.label}
												</span>
											</button>
											<span
												class="ml-auto shrink-0 text-xs text-muted-foreground"
											>
												{groupSelectedCount(subgroup.tests)}/{subgroup.tests
													.length}
											</span>
											<input
												type="checkbox"
												class="shrink-0 accent-primary"
												checked={groupAllSelected(subgroup.tests)}
												indeterminate={groupPartial(subgroup.tests)}
												onchange={() => toggleGroup(subgroup.tests)}
											/>
										</div>
										{#if isOpen('{group.key}/{subgroup.key}')}
											<div class="space-y-0.5 pb-1 pl-8">
												{#each subgroup.tests as test (test.id)}
													<label
														class="flex cursor-pointer items-start gap-2 rounded px-2 py-1 hover:bg-muted/50"
													>
														<input
															type="checkbox"
															class="mt-0.5 shrink-0 accent-primary"
															checked={selectedIds.includes(test.id)}
															onchange={() =>
																form.toggleTestId(test.id)}
														/>
														<div class="min-w-0">
															<div
																class="truncate font-mono text-xs {group
																	.color.text}"
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
													</label>
												{/each}
											</div>
										{/if}
									</div>
								{/each}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	</WithLabel>

	<div class="rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
		<span>{m.Suite()}: <code class="font-mono">{FCAF_SUITE}</code></span>
		<span class="ml-2">· {form.pipelineOutputsCount} evidence sources compiled</span>
	</div>

	<details>
		<summary class="cursor-pointer text-sm font-medium">Advanced configuration</summary>
		<div class="pt-2">
			<CodeEditor lang="yaml" bind:value={form.data.yaml} />
		</div>
	</details>

	{#if form.validationError}
		<p class="text-sm text-destructive">{form.validationError}</p>
	{/if}

	{#if form.intent === 'add'}
		<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
			<T>{m.Add_step()}</T>
		</Button>
	{/if}
</div>
