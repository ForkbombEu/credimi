<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';

	import { resource } from 'runed';

	import A from '@/components/ui-custom/a.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import QrStateful from '@/qr/qr-stateful.svelte';

	import type { PageData } from '../+page';

	import PageSection from '../../../[...path]/_partials/_utils/page-section.svelte';
	import { sections as s } from '../../../[...path]/_partials/_utils/sections';
	import PageLayout from './components/page-layout.svelte';

	//

	type Props = Extract<PageData, { type: 'file-page' }>;

	let { standard, version, suite, file, basePath }: Props = $props();

	//

	const tocSections: IndexItem[] = [s.description, s.qr_code];

	const res = resource(
		() => file,
		// eslint-disable-next-line @typescript-eslint/no-unused-vars
		async (file) => {
			throw new Error('Not implemented');
		}
	);
</script>

<PageLayout {tocSections}>
	{#snippet top()}
		<A class="block" href={basePath}>
			{standard.name} / {version.name} / {suite.name}
		</A>
		<T tag="h1" class="text-balance">{file.replaceAll('+', ' • ')}</T>
	{/snippet}

	{#snippet content()}
		<div class="flex flex-col items-start gap-12 md:flex-row">
			<PageSection indexItem={s.description}>
				<p>{standard.description}</p>
				<p>{suite.description}</p>
			</PageSection>

			<PageSection
				indexItem={s.qr_code}
				class="flex w-full flex-col items-stretch space-y-0 md:w-auto"
			>
				<div class="space-y-4">
					<QrStateful
						src={res.current}
						isLoading={res.loading}
						error={res.error?.message}
						placeholder={m.No_deeplink_available()}
					/>
					{#if res.current}
						<div class="w-60 break-all text-xs">
							<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
							<a href={res.current} target="_blank">{res.current}</a>
						</div>
					{/if}
				</div>
			</PageSection>
		</div>
	{/snippet}
</PageLayout>
