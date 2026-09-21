<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ChevronRightIcon, DownloadIcon, ExternalLinkIcon, SearchIcon } from '@lucide/svelte';
	import LazyImage from '$lib/components/lazy-image.svelte';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';

	import Button from '@/components/ui-custom/button.svelte';

	import {
		checksForTest,
		prepareReportDisplay,
		screenshotsForTest,
		sourceUrl,
		statusClass,
		statusDotClass,
		type CategoryGroup,
		type Report,
		type Screenshot,
		type TestResult
	} from './report.js';

	type Props = {
		report: Report;
		reportUrl: string;
		pdfUrl: string | undefined;
	};

	let { report, reportUrl, pdfUrl }: Props = $props();

	let selectedFilter = $state<string>('all');
	let search = $state('');
	let openGroups = $state<Record<string, boolean>>({});

	const searchQuery = $derived(search.trim().toLowerCase());
	const searching = $derived(searchQuery !== '');

	const display = $derived(
		prepareReportDisplay(report, {
			filter: selectedFilter,
			searchQuery
		})
	);

	function isOpen(key: string): boolean {
		return searching || (openGroups[key] ?? false);
	}

	function toggleOpen(key: string) {
		openGroups[key] = !(openGroups[key] ?? false);
	}

	function expandAll(categories: CategoryGroup[]) {
		for (const category of categories) {
			for (const group of category.groups) openGroups[group.key] = true;
		}
	}

	function collapseAll() {
		openGroups = {};
	}
</script>

{#snippet screenshotGrid(screenshots: Screenshot[])}
	<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
		{#each screenshots as screenshot (screenshot.url)}
			<a
				href={screenshot.url}
				target="_blank"
				rel="noreferrer"
				class="block overflow-hidden rounded border bg-muted/20"
			>
				<LazyImage
					src={screenshot.url}
					alt={screenshot.label}
					class="aspect-video h-auto w-full object-contain"
				/>
				<div class="px-2 py-1 text-xs break-all text-muted-foreground">
					{screenshot.label}
				</div>
			</a>
		{/each}
	</div>
{/snippet}

{#snippet testCard(test: TestResult, testScreenshots: Screenshot[])}
	{@const checks = checksForTest(test)}
	<div class="rounded border p-4">
		<div class="flex items-start justify-between gap-4">
			<div class="min-w-0">
				<div class="font-medium">
					{test.title ?? test.test_id}
				</div>
				{#if sourceUrl(test.test_id)}
					<a
						href={sourceUrl(test.test_id)}
						target="_blank"
						rel="noreferrer"
						class="inline-flex items-center gap-1 text-xs text-primary underline underline-offset-2"
					>
						<code>{test.test_id}</code>
						<ExternalLinkIcon class="size-3" />
					</a>
				{:else}
					<code class="text-xs text-muted-foreground">{test.test_id}</code>
				{/if}
			</div>
			<span
				class="inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium {statusClass(
					test.status
				)}"
			>
				<span
					class="inline-block size-1.5 rounded-full {statusDotClass(test.status)}"
					aria-hidden="true"
				></span>
				{test.status ?? 'unknown'}
			</span>
		</div>
		{#if checks.length}
			<div class="mt-3 space-y-2 border-t pt-3 text-sm">
				{#each checks as validator (validator.id)}
					<div class="flex justify-between gap-3">
						<span>{validator.validator ?? validator.id}</span>
						<strong class={statusClass(validator.status)}
							>{validator.status ?? 'unknown'}</strong
						>
					</div>
					{#if validator.message}
						<div class="text-xs text-muted-foreground">
							{validator.message}
						</div>
					{/if}
				{/each}
			</div>
		{/if}
		{#if testScreenshots.length}
			<div class="mt-4 space-y-2 border-t pt-3">
				<div class="text-xs font-medium tracking-wide text-muted-foreground uppercase">
					Visual evidence
				</div>
				{@render screenshotGrid(testScreenshots)}
			</div>
		{/if}
	</div>
{/snippet}

<div class="pb-6">
	<div class="sticky top-0 z-20 -mx-6 border-b bg-background/95 px-6 py-3 backdrop-blur">
		<div class="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
			<div class="min-w-0 space-y-1.5">
				<div class="flex flex-wrap items-center gap-2">
					<span
						class="inline-flex items-center gap-1.5 rounded-full border bg-muted/40 px-2.5 py-0.5 text-xs font-medium"
					>
						<span
							class="inline-block size-1.5 rounded-full {statusDotClass(
								report.status
							)}"
							aria-hidden="true"
						></span>
						{report.status ?? 'unknown'}
					</span>
					{#if report.suite}
						<span class="text-xs text-muted-foreground">{report.suite}</span>
					{/if}
				</div>
				<div class="flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-xs">
					<span class="text-muted-foreground">{display.totalTests} tests</span>
					<span class="text-green-700">{display.passedTests} passed</span>
					{#if display.failedTests}
						<span class="text-red-700">{display.failedTests} failed</span>
					{/if}
					{#if display.otherTests}
						<span class="text-amber-700">{display.otherTests} other</span>
					{/if}
				</div>
			</div>

			<div class="flex shrink-0 flex-wrap items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					href={reportUrl}
					download="fcaf-assessment.json"
				>
					<DownloadIcon class="size-4" />
					JSON
				</Button>
				{#if pdfUrl}
					<Button size="sm" href={pdfUrl} download="fcaf-assessment.pdf">
						<DownloadIcon class="size-4" />
						PDF
					</Button>
				{/if}
			</div>
		</div>

		{#if display.summaryFilters.length}
			<div class="mt-3 flex flex-wrap items-center gap-1.5">
				<span class="text-xs font-medium text-muted-foreground">Filter:</span>
				<button
					class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors
					{selectedFilter === 'all'
						? 'bg-primary text-primary-foreground'
						: 'border bg-muted/50 text-muted-foreground hover:bg-muted'}"
					onclick={() => (selectedFilter = 'all')}
				>
					All ({display.totalTests})
				</button>
				{#each display.summaryFilters as filter (filter.key)}
					{@const dot = statusDotClass(filter.key)}
					<button
						class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors
						{selectedFilter === filter.key
							? 'bg-primary text-primary-foreground'
							: 'border bg-muted/50 text-muted-foreground hover:bg-muted'}"
						onclick={() =>
							(selectedFilter = selectedFilter === filter.key ? 'all' : filter.key)}
					>
						<span class="inline-block size-2 rounded-full {dot}" aria-hidden="true"
						></span>
						{filter.label} ({filter.count})
					</button>
				{/each}
			</div>
		{/if}
		<div class="mt-3 flex flex-wrap items-center gap-2">
			<div class="relative min-w-0 grow sm:max-w-xs">
				<SearchIcon
					class="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
					aria-hidden="true"
				/>
				<input
					type="search"
					bind:value={search}
					placeholder="Search tests"
					aria-label="Search tests"
					class="h-8 w-full rounded-md border bg-background pr-2 pl-7 text-sm placeholder:text-muted-foreground focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
				/>
			</div>
			<div class="ml-auto flex items-center gap-1">
				<button
					type="button"
					class="rounded-md px-2 py-1 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground"
					onclick={() => expandAll(display.groupedCategories)}
				>
					Expand all
				</button>
				<button
					type="button"
					class="rounded-md px-2 py-1 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground"
					onclick={collapseAll}
				>
					Collapse all
				</button>
			</div>
		</div>

		<div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1.5">
			{#each display.groupedCategories as category (category.category.code)}
				<a
					href="#fcaf-category-{category.category.code}"
					class="inline-flex items-center gap-1.5 rounded-full border bg-background px-2.5 py-1 text-xs font-medium text-foreground hover:bg-muted"
				>
					<span
						class="inline-block size-2 rounded-full {category.category.bar}"
						aria-hidden="true"
					></span>
					<span class="font-medium {category.category.text}"
						>{category.category.label}</span
					>
					<span class="text-muted-foreground">{category.passed}/{category.total}</span>
				</a>
			{/each}
		</div>
	</div>

	<div class="space-y-4 pt-4">
		{#each display.groupedCategories as category (category.category.code)}
			<section
				id="fcaf-category-{category.category.code}"
				class="grid grid-cols-[auto_1fr] overflow-hidden rounded-lg border"
			>
				<div
					class="flex w-9 flex-col items-center gap-2 border-r py-3 {category.category
						.railBg}"
				>
					<span
						class="rotate-180 py-2 font-mono text-[11px] font-semibold tracking-[0.14em] uppercase [writing-mode:vertical-rl] {category
							.category.text}"
					>
						{category.category.label}
					</span>
					<span class="mt-auto font-mono text-[10px] {category.category.text}">
						{category.passed}/{category.total}
					</span>
				</div>

				<div class="min-w-0 divide-y">
					{#each category.groups as group (group.key)}
						<div class="px-4 py-3">
							<div class="flex items-center gap-2">
								<button
									type="button"
									class="flex min-w-0 items-center gap-1.5 text-left"
									aria-expanded={isOpen(group.key)}
									onclick={() => toggleOpen(group.key)}
								>
									<ChevronRightIcon
										class="size-4 shrink-0 text-muted-foreground transition-transform {isOpen(
											group.key
										)
											? 'rotate-90'
											: ''}"
										aria-hidden="true"
									/>
									<span class="truncate text-sm font-medium">{group.label}</span>
								</button>
								<span class="text-xs text-muted-foreground"
									>{group.passed}/{group.total} passed</span
								>
								<span class="ml-auto shrink-0 text-xs text-muted-foreground"
									>{group.total}
									{group.total === 1 ? 'test' : 'tests'}</span
								>
							</div>
							<div class="mt-2 h-1 overflow-hidden rounded-full bg-muted">
								<div
									class="h-full rounded-full {group.rate > 0
										? category.category.bar
										: 'bg-muted'}"
									style="width: {group.rate}%"
								></div>
							</div>

							{#if isOpen(group.key)}
								<div class="mt-3 space-y-3 pl-5">
									{#each group.tests as test (test.test_id)}
										{@render testCard(
											test,
											screenshotsForTest(
												report.presentation?.screenshots ?? [],
												test
											)
										)}
									{/each}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</section>
		{/each}

		{#if display.filteredTests.length === 0 && display.totalTests > 0}
			<p class="py-8 text-center text-sm text-muted-foreground">
				No tests match the current filter.
			</p>
		{/if}
	</div>

	{#if display.unassignedScreenshots.length}
		<div class="space-y-3 border-t py-4">
			<h3 class="font-medium">Other visual evidence</h3>
			{@render screenshotGrid(display.unassignedScreenshots)}
		</div>
	{/if}

	<div class="space-y-3 border-t py-4">
		{#if display.checkedDeeplink}
			<details>
				<summary class="cursor-pointer text-sm font-medium">Checked deeplink</summary>
				<div class="mt-2">
					<code
						class="block overflow-x-auto rounded border bg-muted/20 p-3 text-xs whitespace-nowrap"
					>
						{display.checkedDeeplink}
					</code>
				</div>
			</details>
		{/if}

		<details class="w-full">
			<summary class="cursor-pointer text-sm font-medium">View raw JSON</summary>
			<div class="mt-2 max-h-96 overflow-auto">
				<CodeDisplay content={JSON.stringify(report, null, 2)} language="json" />
			</div>
		</details>
	</div>
</div>
