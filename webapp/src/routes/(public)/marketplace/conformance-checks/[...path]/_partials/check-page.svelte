<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';

	import { setupEWCConnections } from '$lib/wallet-test-pages/ewc.svelte';
	import { WorkflowQrPoller } from '$lib/workflows';
	import { ArrowRightIcon } from 'lucide-svelte';
	import { onMount } from 'svelte';

	import A from '@/components/ui-custom/a.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import RenderMD from '@/components/ui-custom/renderMD.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { localizeHref, m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import type { PageData } from '../+page';

	import PageSection from '../../../[...path]/_partials/_utils/page-section.svelte';
	import { sections as s } from '../../../[...path]/_partials/_utils/sections';
	import EmptyQr from './components/empty-qr.svelte';
	import PageLayout from './components/page-layout.svelte';
	import { startCheck, type StartCheckResult } from './utils';

	//

	type Props = Extract<PageData, { type: 'file-page' }> & { namespace: string | undefined };

	let { standard, version, suite, file, basePath, namespace }: Props = $props();

	//

	const tocSections: IndexItem[] = [s.description, s.qr_code];

	let qrWorkflow = $state<StartCheckResult>();
	let loading = $state(true);
	onMount(async () => {
		if (!$currentUser) return;
		qrWorkflow = await startCheck(standard.uid, version.uid, suite.uid, file);
		loading = false;
	});

	setupEWCConnections(
		() => {
			if (!qrWorkflow) return undefined;
			if (qrWorkflow instanceof Error) return undefined;
			return qrWorkflow.workflowId;
		},
		() => namespace
	);
</script>

<PageLayout {tocSections} logo={suite.logo}>
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
		<EmptyQr>
			<RenderMD
				content={m.conformance_check_qr_code_login_cta({ link: localizeHref('/login') })}
				class="prose-a:text-primary text-balance"
			/>
		</EmptyQr>
	{/if}
{/snippet}

{#snippet loggedQr()}
	{#if loading}
		<EmptyQr>
			<Spinner />
			{m.Loading()}
		</EmptyQr>
	{:else if qrWorkflow instanceof Error}
		<EmptyQr>
			<p>{qrWorkflow.message}</p>
		</EmptyQr>
	{:else if qrWorkflow}
		<WorkflowQrPoller workflowId={qrWorkflow.workflowId} runId={qrWorkflow.runId} />
		<Button
			variant="secondary"
			class="hover:bg-primary/10 border-primary border"
			href="/my/tests/runs/{qrWorkflow.workflowId}/{qrWorkflow.runId}"
		>
			{m.View_check_status()}
			<ArrowRightIcon />
		</Button>
	{/if}
{/snippet}
