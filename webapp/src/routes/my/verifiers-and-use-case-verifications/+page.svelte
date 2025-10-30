<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { appSections } from '$lib/marketplace/sections';
	import { Plus } from 'lucide-svelte';

	import { CollectionManager } from '@/collections-components';
	import CollectionForm from '@/collections-components/form/collectionForm.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../+layout@.svelte';
	import VerifierCard from './verifier-card.svelte';
	import { settings } from './verifier-form-settings.svelte';

	//

	let { data } = $props();
	let { organization } = $derived(data);

	const { verifiers } = appSections;

	setDashboardNavbar({ title: verifiers.label, right: navbarRight });
</script>

<CollectionManager
	collection="verifiers"
	queryOptions={{
		filter: `owner.id = '${organization.id}'`,
		expand: ['use_cases_verifications_via_verifier'],
		sort: ['created', 'DESC']
	}}
	formFieldsOptions={settings}
>
	{#snippet records({ records })}
		<div class="space-y-6">
			{#each records as verifier (verifier)}
				<VerifierCard {verifier} {organization} />
			{/each}
		</div>
	{/snippet}
</CollectionManager>

{#snippet navbarRight()}
	<Sheet title={m.Create_verifier()}>
		{#snippet trigger({ sheetTriggerAttributes })}
			<Button {...sheetTriggerAttributes}>
				<Plus />
				{m.Create_verifier()}
			</Button>
		{/snippet}
		{#snippet content({ closeSheet })}
			<CollectionForm
				collection="verifiers"
				onSuccess={closeSheet}
				fieldsOptions={settings}
			/>
		{/snippet}
	</Sheet>
{/snippet}
