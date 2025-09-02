<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import { ArrowLeft, Share, Trash } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';

	import type { CollectionName } from '@/pocketbase/collections-models';

	import { CollectionForm } from '@/collections-components';
	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { createToggleStore } from '@/components/ui-custom/utils';
	import { m } from '@/i18n';
	import { currentUser, pb } from '@/pocketbase';
	import { type CollectionResponses, type RecordIdString } from '@/pocketbase/types';
	import { getExceptionMessage } from '@/utils/errors';

	import type { RecordProp, TitleProp, TriggerProp } from './types';

	//

	type Props = {
		dialogTitle?: any;
		onAuthorizationAdd?: () => void;
		onAuthorizationRemove?: () => void;
	} & TitleProp &
		TriggerProp &
		RecordProp<C>;

	let {
		record,
		dialogTitle = m.Share_record(),
		onAuthorizationAdd: onAdd = () => {},
		onAuthorizationRemove: onRemove = () => {},
		button: triggerSnippet
	}: Props = $props();

	/* */

	const dialogState = createToggleStore(false);
	const loadingState = createToggleStore(false);

	let formState: 'default' | 'removeAccess' = $state('default');

	async function loadAuthorization(recordId: RecordIdString) {
		const records = await pb
			.collection('authorizations')
			.getFullList({ filter: `record_id="${recordId}"`, requestKey: null });
		return records.at(0);
	}

	function handleSuccess() {
		dialogState.off();
		onAdd();
	}

	//

	let error: unknown = $state();

	async function removeAuthorization(recordId: RecordIdString) {
		try {
			loadingState.on();
			await pb.collection('authorizations').delete(recordId);
			toast.info(m.Record_authorization_removed_successfully());
			formState = 'default';
			loadingState.off();
			dialogState.off();
		} catch (e) {
			error = e;
		}
		onRemove();
	}
	/* Load */

	// TODO - Make a full refactor of how this works, also backend-side

	let authorizationPromise = $derived(loadAuthorization(record.id));
</script>

<Dialog bind:open={$dialogState} title={dialogTitle} contentClass="my-0">
	{#snippet trigger({ props })}
		{#if triggerSnippet}
			{@render triggerSnippet({
				openForm: dialogState.on,
				triggerAttributes: props,
				icon: Share
			})}
		{:else}
			<IconButton variant="outline" icon={Share} {...props} />
		{/if}
	{/snippet}

	{#snippet content()}
		{#await authorizationPromise}
			<div class="flex items-center justify-center self-stretch p-6">
				<Spinner />
			</div>
		{:then authorization}
			{#if formState == 'default'}
				<CollectionForm
					collection="authorizations"
					onSuccess={handleSuccess}
					initialData={authorization}
					recordId={authorization?.id}
					uiOptions={{
						showToastOnSuccess: true,
						toastText: m.Record_shared_successfully()
					}}
					fieldsOptions={{
						hide: {
							record_id: record.id,
							collection_id: record.collectionId,
							owner: $currentUser?.id
						},
						relations: {
							users: {
								displayFields: ['name'],
								queryOptions: { exclude: [$currentUser?.id ?? ''] }
							}
						}
					}}
				>
					{#snippet submitButton({ SubmitButton })}
						<div class="flex justify-between">
							<div>
								{#if authorization}
									<Button
										variant="destructive"
										onclick={() => (formState = 'removeAccess')}
									>
										<Icon src={Trash} mr />
										{m.Remove_access()}
									</Button>
								{/if}
							</div>

							<SubmitButton>
								{#if authorization}
									{m.Update_authorizations()}
								{:else}
									{m.Share()}
								{/if}
							</SubmitButton>
						</div>
					{/snippet}
				</CollectionForm>
			{:else if formState == 'removeAccess' && authorization}
				<T>{m.Are_you_sure_you_want_to_remove_access_to_the_record()}</T>

				{#if error}
					<Alert variant="destructive">
						{#snippet content({ Title, Description })}
							<Title>{m.Error()}</Title>
							<Description>{getExceptionMessage(error)}</Description>
						{/snippet}
					</Alert>
				{/if}

				<div class="mt-4 flex justify-between">
					<Button variant="outline" onclick={() => (formState = 'default')}>
						<Icon src={ArrowLeft} mr />
						{m.Back()}
					</Button>
					<Button
						variant="destructive"
						onclick={() => removeAuthorization(authorization.id)}
					>
						<Icon src={Trash} mr />
						{m.Yes_remove_access()}
					</Button>
				</div>
			{/if}
		{:catch error}
			<Alert variant="destructive">
				{#snippet content({ Title, Description })}
					<Title>{m.Error()}</Title>
					<Description>{getExceptionMessage(error)}</Description>
				{/snippet}
			</Alert>
		{/await}
	{/snippet}
</Dialog>

{#if loadingState}
	<LoadingDialog></LoadingDialog>
{/if}
