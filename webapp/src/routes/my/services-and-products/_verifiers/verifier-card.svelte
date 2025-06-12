<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import {
		CollectionManager,
		RecordCreate,
		RecordDelete,
		RecordEdit
	} from '@/collections-components/manager';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Card from '@/components/ui-custom/card.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import type {
		CredentialsResponse,
		UseCasesVerificationsResponse,
		VerifiersResponse
	} from '@/pocketbase/types';
	import { pb } from '@/pocketbase';
	import { Pencil, Plus } from 'lucide-svelte';
	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { Badge } from '@/components/ui/badge';
	import PublishedStatus from '$lib/layout/published-status.svelte';

	//

	type Props = {
		verifier: VerifiersResponse;
		useCasesVerifications: UseCasesVerificationsResponse[];
		organizationId: string;
	};

	let { verifier, organizationId }: Props = $props();
	const avatarSrc = $derived(pb.files.getURL(verifier, verifier.logo));

	//

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
</script>

<Card class="bg-card" contentClass="space-y-4 p-4">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-4">
			<Avatar src={avatarSrc} fallback={verifier.name} class="rounded-sm border" />
			<div>
				<div class="flex items-center gap-2">
					<T class="font-bold">{verifier.name}</T>
					<PublishedStatus item={verifier} />
				</div>
				<T class="text-xs text-gray-400">{verifier.url}</T>
			</div>
		</div>
		<div>
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
		formFieldsOptions={{
			hide: {
				owner: organizationId,
				verifier: verifier.id
			},
			order: ['name', 'deeplink', 'credentials', 'description', 'published'],
			relations: {
				credentials: {
					mode: 'select',
					displayFields: ['issuer_name', 'name', 'key']
				}
			},
			snippets: {
				description
			}
		}}
	>
		{#snippet top()}
			<div class="flex items-center justify-between pb-1">
				<T class="font-semibold">{m.Verification_use_cases()}</T>
				<RecordCreate>
					{#snippet button({ triggerAttributes })}
						<button
							class="text-primary flex items-center underline hover:cursor-pointer hover:no-underline"
							{...triggerAttributes}
						>
							<Plus size={14} /><span>{m.Add()}</span>
						</button>
					{/snippet}
				</RecordCreate>
			</div>
		{/snippet}
		{#snippet records({ records })}
			<ul class="">
				{#each records as useCaseVerification}
					{@const credentials = useCaseVerification.expand?.credentials ?? []}
					{@const credentialsPreview = credentialsPreviewString(credentials)}
					<li>
						<span>{useCaseVerification.name}</span>
						{#if credentialsPreview}
							<span>({credentialsPreview})</span>
						{/if}
						<PublishedStatus item={useCaseVerification} size="sm" />

						<RecordEdit record={useCaseVerification}>
							{#snippet button({ triggerAttributes })}
								<button
									class="inline translate-y-0.5 rounded-sm p-1 hover:cursor-pointer hover:bg-gray-100"
									{...triggerAttributes}
								>
									<Pencil size={14} />
								</button>
							{/snippet}
						</RecordEdit>
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

{#snippet description({ form }: FieldSnippetOptions<'use_cases_verifications'>)}
	<MarkdownField {form} name="description" />
{/snippet}
