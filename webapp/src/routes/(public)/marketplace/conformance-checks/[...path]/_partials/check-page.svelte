<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';

	import { WorkflowQrPoller } from '$lib/workflows';

	import A from '@/components/ui-custom/a.svelte';
	import RenderMD from '@/components/ui-custom/renderMD.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { localizeHref, m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import type { PageData } from '../+page';

	import PageSection from '../../../[...path]/_partials/_utils/page-section.svelte';
	import { sections as s } from '../../../[...path]/_partials/_utils/sections';
	import PageLayout from './components/page-layout.svelte';

	//

	type Props = Extract<PageData, { type: 'file-page' }>;

	let { standard, version, suite, file, basePath, qrWorkflow }: Props = $props();

	//

	const tocSections: IndexItem[] = [s.description, s.qr_code];
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
				class="flex w-full min-w-60 shrink-0 flex-col items-stretch space-y-0 md:w-auto"
			>
				{@render nruQrCode()}
				{@render loggedQr()}
			</PageSection>
		</div>
	{/snippet}
</PageLayout>

{#snippet nruQrCode()}
	{#if !$currentUser}
		<div
			class={[
				'aspect-square size-60 shrink-0 overflow-hidden rounded-md border',
				'flex items-center justify-center',
				'text-muted-foreground bg-gray-50',
				'text-center text-sm'
			]}
		>
			<RenderMD
				content={m.conformance_check_qr_code_login_cta({ link: localizeHref('/login') })}
				class="prose-a:text-primary text-balance p-3"
			/>
		</div>
	{/if}
{/snippet}

{#snippet loggedQr()}
	{#if qrWorkflow}
		<WorkflowQrPoller workflowId={qrWorkflow.workflowId} runId={qrWorkflow.runId} />
	{/if}
{/snippet}
