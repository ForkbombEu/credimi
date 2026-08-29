<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { DownloadIcon, ExternalLinkIcon } from '@lucide/svelte';
	import LazyImage from '$lib/components/lazy-image.svelte';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import { activeSheet } from '$lib/utils/sheet-state.svelte';
	import { SvelteSet } from 'svelte/reactivity';

	import type { GenericRecord } from '@/utils/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';

	type Props = {
		reportUrl: string | undefined;
		pdfUrl: string | undefined;
		maestroScreenshotUrls: string[];
		sheetTrigger: Snippet<[{ props: GenericRecord; openSheet: () => void }]>;
	};

	type Validator = {
		id?: string;
		status?: string;
		message?: string;
		validator?: string;
	};

	type TestResult = {
		test_id?: string;
		title?: string;
		status?: string;
		assertions?: Array<{ id?: string; status?: string; message?: string; validator?: string }>;
		validators?: Validator[];
	};

	type Report = {
		status?: string;
		suite?: string;
		summary?: Record<string, number>;
		evidence?: Record<string, unknown>;
		executed_tests?: TestResult[];
	};

	type Screenshot = {
		url: string;
		label: string;
	};

	function evidenceScreenshotUrls(value: unknown): string[] {
		const urls: string[] = [];
		const visit = (current: unknown) => {
			if (typeof current === 'string' && /\.(?:png|jpe?g|webp)(?:\?|$)/i.test(current)) {
				urls.push(current);
				return;
			}
			if (Array.isArray(current)) current.forEach(visit);
			else if (current && typeof current === 'object') {
				Object.values(current as Record<string, unknown>).forEach(visit);
			}
		};
		visit(value);
		return [...new Set(urls)];
	}

	function screenshotLabel(url: string): string {
		const filename = decodeURIComponent(url.split('?')[0].split('/').pop() ?? url);
		return filename
			.replace(/\.[^.]+$/, '')
			.replace(/[_-]+/g, ' ')
			.replace(/\s+/g, ' ')
			.trim();
	}

	function uniqueScreenshots(screenshots: Screenshot[]): Screenshot[] {
		const seen = new SvelteSet<string>();
		return screenshots.filter((screenshot) => {
			const key = screenshot.label.toLowerCase();
			if (seen.has(key)) return false;
			seen.add(key);
			return true;
		});
	}

	function screenshotsForTest(screenshots: Screenshot[], test: TestResult): Screenshot[] {
		if (screenshots.length === 0) return [];
		if (screenshots.length === 1) return screenshots;

		const searchable = `${test.test_id ?? ''} ${test.title ?? ''}`.toLowerCase();
		const matching = screenshots.filter(({ label }) => {
			const words = label
				.toLowerCase()
				.split(/\s+/)
				.filter((word) => word.length > 3);
			return words.some((word) => searchable.includes(word));
		});
		return matching;
	}

	function screenshotsWithoutTest(screenshots: Screenshot[], tests: TestResult[]): Screenshot[] {
		if (tests.length <= 1) return [];
		const assigned = new Set(
			tests.flatMap((test) => screenshotsForTest(screenshots, test).map(({ url }) => url))
		);
		return screenshots.filter(({ url }) => !assigned.has(url));
	}

	function fcafSourceUrl(testId: string | undefined): string | undefined {
		if (!testId) return undefined;
		const anchor = testId
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '_')
			.replace(/^_+|_+$/g, '');
		return `https://conformance.eudi.dev/latest-draft/fcaf/suts/wallet_solution/relying_party/ws_rp/#${anchor}`;
	}

	function evidenceDeeplink(value: unknown): string | undefined {
		if (value && typeof value === 'object') {
			for (const [key, nested] of Object.entries(value as Record<string, unknown>)) {
				if (key === 'deeplink' && typeof nested === 'string') return nested;
				const found = evidenceDeeplink(nested);
				if (found) return found;
			}
		}
		if (Array.isArray(value)) {
			for (const nested of value) {
				const found = evidenceDeeplink(nested);
				if (found) return found;
			}
		}
		return undefined;
	}

	function icsFromTestId(testId: string | undefined): string {
		if (!testId) return 'Other';
		const match = testId.match(/^(.+)__\d+$/);
		return match ? match[1] : testId;
	}

	let { reportUrl, pdfUrl, maestroScreenshotUrls, sheetTrigger }: Props = $props();
	let selectedFilter = $state<string>('all');
	let sheetOpen = $state(false);

	$effect(() => {
		if (sheetOpen) {
			activeSheet.open();
		} else {
			activeSheet.close();
		}
	});
	// Fetch lazily: only once the sheet is opened, not for every run row that has a report.
	let cachedReport: Promise<Report | undefined> | undefined;

	const reportPromise = $derived.by(() => {
		if (!reportUrl) return undefined;
		if (!sheetOpen) return cachedReport;
		return (cachedReport ??= fetch(reportUrl)
			.then(async (response) => {
				if (!response.ok) throw new Error(`FCAF report request failed: ${response.status}`);
				return (await response.json()) as Report;
			})
			.catch(() => undefined));
	});

	function statusClass(status = '') {
		if (['passed', 'pass'].includes(status)) return 'text-green-700';
		if (['failed', 'fail', 'error'].includes(status)) return 'text-red-700';
		return 'text-amber-700';
	}

	function statusDotClass(status = '') {
		if (['passed', 'pass'].includes(status)) return 'bg-green-600';
		if (['failed', 'fail', 'error'].includes(status)) return 'bg-red-600';
		return 'bg-amber-500';
	}

	function statusIsPassed(status = '') {
		return (status ?? '').startsWith('pass');
	}

	function statusIsFailed(status = '') {
		return ['failed', 'fail', 'error'].includes(status ?? '');
	}
</script>

{#if reportUrl}
	<Sheet title="FCAF assessment" class="sm:max-w-3xl" bind:open={sheetOpen}>
		{#snippet trigger({ sheetTriggerAttributes: props, openSheet })}
			{@render sheetTrigger({ props, openSheet })}
		{/snippet}
		{#snippet content()}
			{#await reportPromise then report}
				<div class="pb-6">
					{#if report}
						{@const evidenceScreenshots = evidenceScreenshotUrls(report)}
						{@const allScreenshots = uniqueScreenshots(
							[...new Set([...maestroScreenshotUrls, ...evidenceScreenshots])].map(
								(url) => ({
									url,
									label: screenshotLabel(url)
								})
							)
						)}
						{@const executedTests = report.executed_tests ?? []}
						{@const totalTests = executedTests.length}
						{@const passedTests = executedTests.filter((t) =>
							statusIsPassed(t.status)
						).length}
						{@const failedTests = executedTests.filter((t) =>
							statusIsFailed(t.status)
						).length}
						{@const otherTests = totalTests - passedTests - failedTests}
						{@const unassignedScreenshots = screenshotsWithoutTest(
							allScreenshots,
							executedTests
						)}

						<div
							class="sticky top-0 z-20 -mx-6 border-b bg-background/95 px-6 py-3 backdrop-blur"
						>
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
											<span class="text-xs text-muted-foreground"
												>{report.suite}</span
											>
										{/if}
									</div>
									<div
										class="flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-xs"
									>
										<span class="text-muted-foreground">{totalTests} tests</span
										>
										<span class="text-green-700">{passedTests} passed</span>
										{#if failedTests}
											<span class="text-red-700">{failedTests} failed</span>
										{/if}
										{#if otherTests}
											<span class="text-amber-700">{otherTests} other</span>
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
										<Button
											size="sm"
											href={pdfUrl}
											download="fcaf-assessment.pdf"
										>
											<DownloadIcon class="size-4" />
											PDF
										</Button>
									{/if}
								</div>
							</div>

							{#if report.summary}
								{@const entries = Object.entries(report.summary).filter(
									([, c]) => c > 0
								)}
								<div class="mt-3 flex flex-wrap items-center gap-1.5">
									<span class="text-xs font-medium text-muted-foreground"
										>Filter:</span
									>
									<button
										class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors
										{selectedFilter === 'all'
											? 'bg-primary text-primary-foreground'
											: 'border bg-muted/50 text-muted-foreground hover:bg-muted'}"
										onclick={() => (selectedFilter = 'all')}
									>
										All ({totalTests})
									</button>
									{#each entries as [label, count] (label)}
										{@const dot = statusDotClass(label)}
										<button
											class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors
											{selectedFilter === label
												? 'bg-primary text-primary-foreground'
												: 'border bg-muted/50 text-muted-foreground hover:bg-muted'}"
											onclick={() =>
												(selectedFilter =
													selectedFilter === label ? 'all' : label)}
										>
											<span
												class="inline-block size-2 rounded-full {dot}"
												aria-hidden="true"
											></span>
											{label} ({count})
										</button>
									{/each}
								</div>
							{/if}
						</div>

						{@const filteredTests = executedTests.filter(
							(t) =>
								selectedFilter === 'all' ||
								(t.status ?? '').startsWith(selectedFilter)
						)}
						{@const grouped = filteredTests.reduce(
							(acc, t) => {
								const key = icsFromTestId(t.test_id);
								(acc[key] ??= []).push(t);
								return acc;
							},
							{} as Record<string, TestResult[]>
						)}
						<div class="space-y-6 pt-4">
							{#each Object.entries(grouped) as [ics, tests] (ics)}
								{@const icsPassed = (tests ?? []).filter((t) =>
									statusIsPassed(t.status)
								).length}
								{@const icsTotal = (tests ?? []).length}
								{@const icsRate =
									icsTotal > 0 ? Math.round((icsPassed / icsTotal) * 100) : 0}
								<section>
									<div class="mb-2 flex items-baseline gap-2">
										<h3 class="font-mono text-sm font-semibold break-all">
											{ics}
										</h3>
										<span class="text-xs text-muted-foreground">
											{icsPassed}/{icsTotal} passed
										</span>
									</div>
									<div class="mb-3 h-1 overflow-hidden rounded-full bg-muted">
										<div
											class="h-full rounded-full {icsTotal > 0
												? icsPassed === icsTotal
													? 'bg-green-600'
													: icsPassed > 0
														? 'bg-amber-500'
														: 'bg-red-600'
												: 'bg-muted'}"
											style="width: {icsRate}%"
										></div>
									</div>
									<div class="space-y-3">
										{#each tests ?? [] as test (test.test_id)}
											{@const testScreenshots = screenshotsForTest(
												allScreenshots,
												test
											)}
											<div class="rounded border p-4">
												<div class="flex items-start justify-between gap-4">
													<div class="min-w-0">
														<div class="font-medium">
															{test.title ?? test.test_id}
														</div>
														{#if fcafSourceUrl(test.test_id)}
															<a
																href={fcafSourceUrl(test.test_id)}
																target="_blank"
																rel="noreferrer"
																class="inline-flex items-center gap-1 text-xs text-primary underline underline-offset-2"
															>
																<code>{test.test_id}</code>
																<ExternalLinkIcon class="size-3" />
															</a>
														{:else}
															<code
																class="text-xs text-muted-foreground"
																>{test.test_id}</code
															>
														{/if}
													</div>
													<span
														class="inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium {statusClass(
															test.status
														)}"
													>
														<span
															class="inline-block size-1.5 rounded-full {statusDotClass(
																test.status
															)}"
															aria-hidden="true"
														></span>
														{test.status ?? 'unknown'}
													</span>
												</div>
												{#if test.assertions?.length || test.validators?.length}
													<div
														class="mt-3 space-y-2 border-t pt-3 text-sm"
													>
														{#each test.validators ?? test.assertions ?? [] as validator (validator.id)}
															<div class="flex justify-between gap-3">
																<span
																	>{validator.validator ??
																		validator.id}</span
																>
																<strong
																	class={statusClass(
																		validator.status
																	)}
																	>{validator.status ??
																		'unknown'}</strong
																>
															</div>
															{#if validator.message}<div
																	class="text-xs text-muted-foreground"
																>
																	{validator.message}
																</div>{/if}
														{/each}
													</div>
												{/if}
												{#if testScreenshots.length}
													<div class="mt-4 space-y-2 border-t pt-3">
														<div
															class="text-xs font-medium tracking-wide text-muted-foreground uppercase"
														>
															Visual evidence
														</div>
														<div
															class="grid grid-cols-2 gap-2 sm:grid-cols-4"
														>
															{#each testScreenshots as screenshot (screenshot.url)}
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
																	<div
																		class="px-2 py-1 text-xs break-all text-muted-foreground"
																	>
																		{screenshot.label}
																	</div>
																</a>
															{/each}
														</div>
													</div>
												{/if}
											</div>
										{/each}
									</div>
								</section>
							{/each}
						</div>

						{#if unassignedScreenshots.length}
							<div class="space-y-3 border-t py-4">
								<h3 class="font-medium">Other visual evidence</h3>
								<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
									{#each unassignedScreenshots as screenshot (screenshot.url)}
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
											<div
												class="px-2 py-1 text-xs break-all text-muted-foreground"
											>
												{screenshot.label}
											</div>
										</a>
									{/each}
								</div>
							</div>
						{/if}

						{@const checkedDeeplink = evidenceDeeplink(report.evidence)}
						<div class="space-y-3 border-t py-4">
							{#if checkedDeeplink}
								<details>
									<summary class="cursor-pointer text-sm font-medium"
										>Checked deeplink</summary
									>
									<div class="mt-2">
										<code
											class="block overflow-x-auto rounded border bg-muted/20 p-3 text-xs whitespace-nowrap"
										>
											{checkedDeeplink}
										</code>
									</div>
								</details>
							{/if}

							<details class="w-full">
								<summary class="cursor-pointer text-sm font-medium"
									>View raw JSON</summary
								>
								<div class="mt-2 max-h-96 overflow-auto">
									<CodeDisplay
										content={JSON.stringify(report, null, 2)}
										language="json"
									/>
								</div>
							</details>
						</div>
					{:else}
						<p class="py-8 text-sm text-muted-foreground">
							Unable to load the FCAF assessment.
						</p>
					{/if}
				</div>
			{/await}
		{/snippet}
	</Sheet>
{/if}
