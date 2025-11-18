<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { CopyPlus } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';

	import type { CollectionResponses } from '@/pocketbase/types';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';
	import { cloneRecord } from '@/pocketbase/utils';

	type CloneableCollections =
		| 'wallet_actions'
		| 'credentials'
		| 'use_cases_verifications'
		| 'pipelines';

	//

	type Props = {
		record: CollectionResponses[CloneableCollections];
		collectionName: CloneableCollections;
		onSuccess?: () => void;
		/** Custom button renderer */
		button?: Snippet<[{ triggerAttributes: object; icon: typeof CopyPlus }]>;
	};

	const { record, collectionName, onSuccess = () => {}, button }: Props = $props();

	//

	let isCloning = $state(false);

	async function handleClone() {
		if (isCloning) return;

		isCloning = true;

		try {
			await cloneRecord(collectionName, record.id);
			toast.success(`${getCollectionDisplayName(collectionName)} cloned successfully`);
			onSuccess();
		} catch (error) {
			console.error('Error cloning record:', error);
			toast.error(`Failed to clone ${getCollectionDisplayName(collectionName)}`);
		} finally {
			isCloning = false;
		}
	}

	function getCollectionDisplayName(collectionName: CloneableCollections): string {
		switch (collectionName) {
			case 'wallet_actions':
				return 'Wallet Action';
			case 'credentials':
				return 'Credential';
			case 'use_cases_verifications':
				return 'Verification Use Case';
			case 'pipelines':
				return 'Pipeline';
			default:
				return 'Record';
		}
	}
</script>

{#if button}
	{@render button({ triggerAttributes: { onclick: handleClone }, icon: CopyPlus })}
{:else}
	<Tooltip>
		<IconButton
			variant="outline"
			size="sm"
			icon={CopyPlus}
			disabled={isCloning}
			onclick={handleClone}
			class="h-8 w-8 p-0"
		/>
		{#snippet content()}
			<p>{m.Clone()}</p>
		{/snippet}
	</Tooltip>
{/if}
