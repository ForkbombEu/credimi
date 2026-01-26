<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import DashboardCardManagerTop from '$lib/layout/dashboard-card-manager-top.svelte';
	import DashboardCardManagerUI from '$lib/layout/dashboard-card-manager-ui.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';

	import type {
		OrganizationsResponse,
		UseCasesVerificationsResponse,
		VerifiersResponse
	} from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components/manager';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { options } from './use-case-verification-form-options.svelte';

	//

	type Props = {
		verifier: VerifiersResponse;
		organization: OrganizationsResponse;
	};

	let { verifier = $bindable(), organization }: Props = $props();

	//

	const managerOptions = options(organization.id, verifier.id);

	function getVerificationPublicUrl(ucv: UseCasesVerificationsResponse) {
		return `/marketplace/use_cases_verifications/${organization.canonified_name}/${verifier.canonified_name}/${ucv.canonified_name}`;
	}
</script>

<DashboardCard
	record={verifier}
	avatar={(v) => pb.files.getURL(v, v.logo)}
	links={{
		URL: verifier.url
	}}
>
	{#snippet content()}
		{@render useCasesVerificationsManager()}
	{/snippet}
</DashboardCard>

{#snippet useCasesVerificationsManager()}
	<CollectionManager
		collection="use_cases_verifications"
		queryOptions={{
			filter: `verifier = '${verifier.id}' && owner.id = '${organization.id}'`,
			expand: ['credentials']
		}}
		formRefineSchema={managerOptions.refineSchema}
		formFieldsOptions={managerOptions.fieldsOptions}
		hide={['empty_state']}
	>
		{#snippet top()}
			<DashboardCardManagerTop
				label={m.Verification_use_cases()}
				buttonText={m.Add_a_verification_use_case()}
				recordCreateOptions={{
					formTitle: `${m.Verifier()}: ${verifier.name} — ${m.Add_a_verification_use_case()}`
				}}
			/>
		{/snippet}

		{#snippet records({ records })}
			<DashboardCardManagerUI
				{records}
				nameField="name"
				publicUrl={getVerificationPublicUrl}
			/>
		{/snippet}
	</CollectionManager>
{/snippet}
