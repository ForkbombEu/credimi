<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { pb } from '@/pocketbase/index.js';
	import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';
	import { partitionPromises } from '@/utils/promise';

	import { pageDetails } from './_utils/types';

	export async function getVerifierDetails(itemId: string, fetchFn = fetch) {
		const verifier = await new PocketbaseQueryAgent(
			{
				collection: 'verifiers',
				expand: ['use_cases_verifications_via_verifier']
			},
			{ fetch: fetchFn }
		).getOne(itemId);

		const useCasesVerifications = verifier.expand?.use_cases_verifications_via_verifier ?? [];

		const [marketplaceUseCasesVerifications] = await partitionPromises(
			useCasesVerifications.map((v) =>
				pb.collection('marketplace_items').getOne(v.id, { fetch })
			)
		);

		const [marketplaceCredentials] = await partitionPromises(
			useCasesVerifications
				.flatMap((v) => v.credentials)
				.map((c) => pb.collection('marketplace_items').getOne(c, { fetch }))
		);

		return pageDetails('verifiers', {
			verifier,
			marketplaceCredentials,
			marketplaceUseCasesVerifications
		});
	}
</script>

<script lang="ts">
	import InfoBox from '$lib/layout/infoBox.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import { MarketplaceItemCard } from '$marketplace/_utils/index.js';
	import { settings } from '$routes/my/services-and-products/_verifiers/verifier-form-settings.svelte';
	import { String } from 'effect';

	import { CollectionForm } from '@/collections-components/index.js';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import DescriptionSection from './_utils/description-section.svelte';
	import EditSheet from './_utils/edit-sheet.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import PageSection from './_utils/page-section.svelte';
	import { sections as s } from './_utils/sections';

	//

	type Props = Awaited<ReturnType<typeof getVerifierDetails>>;
	let { verifier, marketplaceCredentials, marketplaceUseCasesVerifications }: Props = $props();

	//

	const standardAndVersion = $derived(verifier.standard_and_version.split(','));
</script>

<LayoutWithToc sections={[s.general_info, s.description, s.credentials, s.use_case_verifications]}>
	<PageSection indexItem={s.general_info}>
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			<InfoBox label="URL" url={verifier.url} copyable={true} />

			{#if String.isNonEmpty(verifier.repository_url)}
				<InfoBox label="Homepage" url={verifier.repository_url} copyable={true} />
			{:else}
				<div></div>
			{/if}

			<InfoBox label={m.Standard_and_version()}>
				<T>{standardAndVersion.join(', ')}</T>
			</InfoBox>

			<InfoBox label={m.Signing_algorithms_supported()}>
				<T>{verifier.signing_algorithms.join(', ')}</T>
			</InfoBox>

			<InfoBox label={m.Cryptographic_binding_methods_supported()}>
				<T>{verifier.cryptographic_binding_methods.join(', ')}</T>
			</InfoBox>

			<InfoBox label={m.Credentials_format()}>
				<T>{verifier.format.join(', ')}</T>
			</InfoBox>
		</div>
	</PageSection>

	<DescriptionSection description={verifier.description} />

	<PageSection indexItem={s.credentials} empty={marketplaceCredentials.length === 0}>
		<PageGrid>
			{#each marketplaceCredentials as credential (credential.id)}
				<MarketplaceItemCard item={credential} />
			{/each}
		</PageGrid>
	</PageSection>

	<PageSection
		indexItem={s.use_case_verifications}
		empty={marketplaceUseCasesVerifications.length === 0}
	>
		<PageGrid>
			{#each marketplaceUseCasesVerifications as useCaseVerification (useCaseVerification.id)}
				<MarketplaceItemCard item={useCaseVerification} />
			{/each}
		</PageGrid>
	</PageSection>
</LayoutWithToc>

<EditSheet>
	{#snippet children({ closeSheet })}
		<T tag="h2" class="mb-4">{m.Edit()} {verifier.name}</T>
		<CollectionForm
			collection="verifiers"
			recordId={verifier.id}
			initialData={verifier}
			fieldsOptions={settings}
			onSuccess={closeSheet}
			uiOptions={{
				showToastOnSuccess: true
			}}
		>
			{#snippet submitButtonContent()}
				{m.Save()}
			{/snippet}
		</CollectionForm>
	{/snippet}
</EditSheet>
