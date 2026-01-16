<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import { Trash, X } from '@lucide/svelte';
	import { onDestroy } from 'svelte';
	import { toast } from 'svelte-sonner';

	import type { CollectionName } from '@/pocketbase/collections-models';
	import type { CollectionResponses } from '@/pocketbase/types';
	import type { MaybePromise } from '@/utils/types';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { createToggleStore } from '@/components/ui-custom/utils';
	import { m } from '@/i18n';
	import { getExceptionMessage } from '@/utils/errors';

	import type { TitleProp, TriggerProp } from './types';

	import { getCollectionManagerContext } from '../collectionManagerContext';

	//

	type Props = {
		record: CollectionResponses[C];
		beforeDelete?: (record: CollectionResponses[C]) => MaybePromise<void>;
	} & TriggerProp &
		TitleProp;

	const {
		record,
		formTitle = m.Delete_record(),
		beforeDelete = () => {},
		button: triggerSnippet
	}: Props = $props();

	const { manager } = $derived(getCollectionManagerContext());

	//

	const dialogState = createToggleStore(false);

	let error: string | undefined = $state(undefined);

	async function deleteRecord(record: CollectionResponses[C]) {
		error = undefined;
		try {
			await beforeDelete(record);
			await manager.recordService.delete(record.id);
			dialogState.off();
			toast.info(m.Record_deleted_successfully());
		} catch (e) {
			error = getExceptionMessage(e);
		}
	}

	onDestroy(() => {
		error = undefined;
	});
</script>

<Dialog bind:open={$dialogState} title={formTitle} onclose={() => (error = undefined)}>
	{#snippet trigger({ props })}
		{#if triggerSnippet}
			{@render triggerSnippet({
				openForm: dialogState.on,
				triggerAttributes: props,
				icon: Trash
			})}
		{:else}
			<IconButton icon={Trash} {...props} />
		{/if}
	{/snippet}

	{#snippet content()}
		<div class="space-y-6">
			<T>{m.Are_you_sure_you_want_to_delete_this_record()}</T>

			{#if error}
				<Alert variant="destructive">
					<p class="font-bold">{m.Error()}</p>
					<p>{error}</p>
				</Alert>
			{/if}

			<div class="flex justify-center gap-2">
				<Button variant="destructive" onclick={() => deleteRecord(record)}>
					<Icon src={Trash} mr />
					{m.Delete()}
				</Button>

				<Button variant="outline" onclick={dialogState.off}>
					<Icon src={X} mr />
					{m.Cancel()}
				</Button>
			</div>
		</div>
	{/snippet}
</Dialog>
