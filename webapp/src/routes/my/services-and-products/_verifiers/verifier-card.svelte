<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Eye, EyeOff, Plus } from 'lucide-svelte';

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
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Card from '@/components/ui-custom/card.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import SwitchWithIcons from '@/components/ui-custom/switch-with-icons.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { options } from './use-case-verification-form-options.svelte';

	//

	type Props = {
		verifier: VerifiersResponse;
		useCasesVerifications: UseCasesVerificationsResponse[];
		organizationId: string;
		organization?: OrganizationsResponse;
	};

	let { verifier = $bindable(), organizationId, organization }: Props = $props();
	const avatarSrc = $derived(pb.files.getURL(verifier, verifier.logo));

	//

	function getUseCaseVerificationCopyText(useCaseVerification: UseCasesVerificationsResponse) {
		const organizationName =
			organization?.canonified_name ||
			organization?.name ||
			organizationId ||
			'Unknown Organization';
		const verifierName = verifier.canonified_name || verifier.name || 'Unknown Verifier';
		const useCaseName =
			useCaseVerification.canonified_name ||
			useCaseVerification.name ||
			'Unknown Use Case Verification';

		return `${organizationName}/${verifierName}/${useCaseName}`;
	}

	async function updatePublished(recordId: string, published: boolean) {
		const res = await pb.collection('verifiers').update(recordId, { published });
		verifier.published = res.published;
	}

	async function updateUseCasePublished(
		recordId: string,
		published: boolean,
		onSuccess: () => void
	) {
		await pb.collection('use_cases_verifications').update(recordId, {
			published
		});
		onSuccess();
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

<Card class="bg-card" contentClass="space-y-4 p-4">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-4">
			<Avatar src={avatarSrc} fallback={verifier.name} class="rounded-sm border" />
			<div>
				<div class="flex items-center gap-2">
					<T class="font-bold">{verifier.name}</T>
				</div>
				<T class="text-xs text-gray-400">{verifier.url}</T>
			</div>
		</div>
		<div class="flex items-center gap-1">
			<SwitchWithIcons
				offIcon={EyeOff}
				onIcon={Eye}
				size="md"
				checked={verifier.published}
				onCheckedChange={() => updatePublished(verifier.id, !verifier.published)}
			/>
			<RecordEdit record={verifier} />
			<RecordDelete record={verifier} />
		</div>
	</div>

	<Separator />

	<div class="space-y-0.5 text-sm">
		{@render useCasesVerificationsList()}
	</div>
</Card>

{#snippet useCasesVerificationsList()}
	<CollectionManager
		collection="use_cases_verifications"
		queryOptions={{
			filter: `verifier = '${verifier.id}' && owner.id = '${organizationId}'`,
			expand: ['credentials']
		}}
		formFieldsOptions={options(organizationId, verifier.id)}
	>
		{#snippet top()}
			<div class="flex items-center justify-between pb-1">
				<T class="font-semibold">{m.Verification_use_cases()}</T>
				<RecordCreate>
					{#snippet button({ triggerAttributes })}
						<button
							type="button"
							class="text-primary flex items-center underline hover:cursor-pointer hover:no-underline"
							{...triggerAttributes}
						>
							<Plus size={14} /><span>{m.Add()}</span>
						</button>
					{/snippet}
				</RecordCreate>
			</div>
		{/snippet}

		{#snippet records({ records, reloadRecords })}
			<ul class="space-y-2">
				{#each records as useCaseVerification}
					{@const credentials = useCaseVerification.expand?.credentials ?? []}
					{@const credentialsPreview = credentialsPreviewString(credentials)}
					<li class="bg-muted flex items-center justify-between rounded-md p-2 pl-3 pr-2">
						<div class="flex-1">
							<span>{useCaseVerification.name}</span>
							{#if credentialsPreview}
								<span class="text-gray-500">({credentialsPreview})</span>
							{/if}
						</div>

						<div class="flex items-center gap-1">
							<Tooltip>
								<CopyButtonSmall
									textToCopy={getUseCaseVerificationCopyText(useCaseVerification)}
									square
								/>
								{#snippet content()}
									<p>{copyUseCaseVerificationTooltipText}</p>
								{/snippet}
							</Tooltip>

							<SwitchWithIcons
								offIcon={EyeOff}
								onIcon={Eye}
								size="sm"
								checked={useCaseVerification.published}
								disabled={!verifier.published}
								onCheckedChange={() =>
									updateUseCasePublished(
										useCaseVerification.id,
										!useCaseVerification.published,
										reloadRecords
									)}
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

		{#snippet emptyState()}
			<div class="rounded-sm border p-1">
				<T class="text-center text-gray-400">{m.Add_a_verification_use_case()}</T>
			</div>
		{/snippet}
	</CollectionManager>
{/snippet}
