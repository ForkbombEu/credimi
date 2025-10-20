<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BlueButton from '$lib/layout/blue-button.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import LabelLink from '$lib/layout/label-link.svelte';
	import PublishedSwitch from '$lib/layout/published-switch.svelte';

	import type {
		CredentialsResponse,
		OrganizationsResponse,
		UseCasesVerificationsResponse,
		VerifiersResponse
	} from '@/pocketbase/types';

	import {
		CollectionManager,
		RecordClone,
		RecordCreate,
		RecordDelete,
		RecordEdit
	} from '@/collections-components/manager';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { options } from './use-case-verification-form-options.svelte';

	//

	type Props = {
		verifier: VerifiersResponse;
		useCasesVerifications: UseCasesVerificationsResponse[];
		organization: OrganizationsResponse;
	};

	let { verifier = $bindable(), organization }: Props = $props();

	//

	function getUseCaseVerificationCopyText(useCaseVerification: UseCasesVerificationsResponse) {
		const organizationName =
			organization.canonified_name ||
			organization.name ||
			organization.id ||
			'Unknown Organization';
		const verifierName = verifier.canonified_name || verifier.name || 'Unknown Verifier';
		const useCaseName =
			useCaseVerification.canonified_name ||
			useCaseVerification.name ||
			'Unknown Use Case Verification';

		return `${organizationName}/${verifierName}/${useCaseName}`;
	}

	function credentialsPreviewString(credentials: CredentialsResponse[]): string | undefined {
		if (credentials.length === 0) return undefined;
		let preview = '';
		if (credentials.length === 1) {
			preview = `${credentials[0].name}`;
		} else if (credentials.length === 2) {
			preview = `${credentials[0].name}, ${credentials[1].name}`;
		} else {
			preview = `${credentials[0].name}, ${credentials[1].name} and ${credentials.length - 2} others`;
		}
		return preview;
	}

	const copyUseCaseVerificationTooltipText = `${m.Copy()} ${m.Organization()}/${m.Verifier()}/${m.Use_case_verification()}`;
</script>

<DashboardCard
	record={verifier}
	{organization}
	avatar={(v) => pb.files.getURL(v, v.logo)}
	links={{
		URL: verifier.url
	}}
>
	{#snippet content()}
		{@render useCasesVerificationsList()}
	{/snippet}
</DashboardCard>

{#snippet useCasesVerificationsList()}
	{@const opts = options(organization.id, verifier.id)}
	<CollectionManager
		collection="use_cases_verifications"
		queryOptions={{
			filter: `verifier = '${verifier.id}' && owner.id = '${organization.id}'`,
			expand: ['credentials']
		}}
		formRefineSchema={opts.refineSchema}
		formFieldsOptions={opts.fieldsOptions}
		hide={['empty_state']}
	>
		{#snippet top()}
			<div class="flex items-center justify-between">
				<T class="font-semibold">{m.Verification_use_cases()}</T>
				<RecordCreate>
					{#snippet button({ triggerAttributes, icon })}
						<BlueButton
							{icon}
							text={m.Add_a_verification_use_case()}
							{...triggerAttributes}
						/>
					{/snippet}
				</RecordCreate>
			</div>
		{/snippet}

		{#snippet records({ records, reloadRecords })}
			<ul class="space-y-2 pt-1">
				{#each records as useCaseVerification (useCaseVerification.id)}
					{@const credentials = useCaseVerification.expand?.credentials ?? []}
					{@const credentialsPreview = credentialsPreviewString(credentials)}
					<li class="bg-muted flex items-center justify-between rounded-md p-2 pl-3 pr-2">
						<div class="flex-1">
							<LabelLink
								label={useCaseVerification.name}
								href="/marketplace/use_cases_verifications/{organization?.canonified_name}/{useCaseVerification.canonified_name}"
								published={useCaseVerification.published}
							/>
							{#if credentialsPreview}
								<span class="text-gray-500">({credentialsPreview})</span>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							<Tooltip>
								<CopyButtonSmall
									textToCopy={getUseCaseVerificationCopyText(useCaseVerification)}
									square
								/>
								{#snippet content()}
									<p>{copyUseCaseVerificationTooltipText}</p>
								{/snippet}
							</Tooltip>

							<PublishedSwitch
								record={useCaseVerification}
								size="sm"
								field="published"
							/>
							<RecordClone
								collectionName="use_cases_verifications"
								record={useCaseVerification}
								onSuccess={reloadRecords}
							/>
							<RecordEdit record={useCaseVerification}>
								{#snippet button({ triggerAttributes, icon })}
									<IconButton
										size="sm"
										variant="outline"
										{icon}
										{...triggerAttributes}
									/>
								{/snippet}
							</RecordEdit>
							<RecordDelete record={useCaseVerification}>
								{#snippet button({ triggerAttributes, icon })}
									<IconButton
										size="sm"
										variant="outline"
										{icon}
										{...triggerAttributes}
									/>
								{/snippet}
							</RecordDelete>
						</div>
					</li>
				{/each}
			</ul>
		{/snippet}
	</CollectionManager>
{/snippet}
