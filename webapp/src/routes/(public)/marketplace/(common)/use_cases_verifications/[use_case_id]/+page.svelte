<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import MarketplacePageLayout from '$lib/layout/marketplace-page-layout.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import { generateDeeplinkFromYaml } from '$lib/utils';
	import { generateMarketplaceSection } from '$marketplace/_utils/index.js';
	import MarketplaceItemCard from '$marketplace/_utils/marketplace-item-card.svelte';
	import { options } from '$routes/my/services-and-products/_verifiers/use-case-verification-form-options.svelte';
	import { onMount } from 'svelte';

	import CollectionForm from '@/collections-components/form/collectionForm.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n/index.js';
	import QrStateful from '@/qr/qr-stateful.svelte';

	import EditSheet from '../../_utils/edit-sheet.svelte';

	//

	let { data } = $props();
	const { useCaseVerification } = $derived(data);
	let qrLink = $state<string>('');
	let isProcessingYaml = $state(false);
	let yamlProcessingError = $state(false);

	//

	const sections = generateMarketplaceSection('use_cases_verifications', {
		hasRelatedVerifier: true,
		hasRelatedCredentials: true
	});

	onMount(async () => {
		if (useCaseVerification.deeplink) {
			isProcessingYaml = true;
			yamlProcessingError = false;
			try {
				const result = await generateDeeplinkFromYaml(useCaseVerification.deeplink);
				if (result.deeplink) {
					qrLink = result.deeplink;
				}
			} catch (error) {
				console.error('Failed to process YAML for credential offer:', error);
				yamlProcessingError = true;
			} finally {
				isProcessingYaml = false;
			}
		}
	});
</script>

<MarketplacePageLayout tableOfContents={sections}>
	<div class="flex items-start gap-6">
		<div class="grow space-y-6">
			<PageHeader title={sections.general_info.label} id={sections.general_info.anchor} />

			<div class="prose">
				<RenderMd content={data.useCaseVerification.description} />
			</div>
		</div>

		<div class="flex flex-col items-stretch">
			<PageHeader title={m.QR_code()} id="qr" />
			<QrStateful
				src={qrLink}
				isLoading={isProcessingYaml}
				error={yamlProcessingError ? 'Dynamic generation failed' : undefined}
				loadingText="Processing YAML configuration..."
				placeholder="No credential offer available"
			/>
			<div class="w-60 break-all pt-4 text-xs">
				<a href={qrLink} target="_self">{qrLink}</a>
			</div>
		</div>
	</div>

	<div class="flex w-full flex-col gap-6 sm:flex-row">
		<div class="shrink-0 grow basis-1">
			<PageHeader
				title={sections.related_verifier.label}
				id={sections.related_verifier.anchor}
			/>
			<MarketplaceItemCard item={data.verifierMarketplaceItem} />
		</div>

		<div class="shrink-0 grow basis-1">
			<PageHeader
				title={sections.related_credentials.label}
				id={sections.related_credentials.anchor}
			/>

			<div class="flex flex-col gap-2">
				{#each data.marketplaceCredentials as marketplaceCredential}
					<MarketplaceItemCard item={marketplaceCredential} />
				{/each}
			</div>
		</div>
	</div>
</MarketplacePageLayout>

<EditSheet>
	{#snippet children({ closeSheet })}
		<T tag="h2" class="mb-4">{m.Edit()} {useCaseVerification.name}</T>
		<CollectionForm
			collection="use_cases_verifications"
			recordId={useCaseVerification.id}
			initialData={useCaseVerification}
			fieldsOptions={options(useCaseVerification.owner, useCaseVerification.verifier)}
			onSuccess={closeSheet}
		/>
	{/snippet}
</EditSheet>
