<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps, Snippet } from 'svelte';

	import { runWithLoading } from '$lib/utils';
	import { CopyPlus } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	import type { CollectionName } from '@/pocketbase/collections-models';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	type Props = {
		recordId: string;
		collectionName: CollectionName;
		button?: Snippet<[{ triggerAttributes: object; icon: typeof CopyPlus }]>;
		size?: ComponentProps<typeof IconButton>['size'];
	};

	const { recordId, collectionName, button, size = 'sm' }: Props = $props();

	//

	let isCloning = $state(false);

	async function handleClone() {
		if (isCloning) return;
		isCloning = true;
		runWithLoading({
			fn: async () => {
				try {
					await pb.send('/api/clone-record', {
						method: 'POST',
						body: {
							id: recordId,
							collection: collectionName
						}
					});
					toast.success(
						m.data_cloned_successfully({
							data_type: getCollectionDisplayName(collectionName)
						})
					);
				} catch {
					toast.error(
						m.failed_to_clone_data({
							data_type: getCollectionDisplayName(collectionName)
						})
					);
				} finally {
					isCloning = false;
				}
			},
			showSuccessToast: false
		});
	}

	function getCollectionDisplayName(collectionName: CollectionName): string {
		switch (collectionName) {
			case 'wallet_actions':
				return m.Wallet_action();
			case 'credentials':
				return m.Credential();
			case 'use_cases_verifications':
				return m.Verification_use_case();
			case 'pipelines':
				return m.Pipeline();
			default:
				return m.record();
		}
	}
</script>

{#if button}
	{@render button({ triggerAttributes: { onclick: handleClone }, icon: CopyPlus })}
{:else}
	<Tooltip>
		<IconButton
			variant="outline"
			{size}
			icon={CopyPlus}
			disabled={isCloning}
			onclick={handleClone}
		/>
		{#snippet content()}
			<p>{m.Clone()}</p>
		{/snippet}
	</Tooltip>
{/if}
