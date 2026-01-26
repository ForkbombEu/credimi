<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import Pencil from '@lucide/svelte/icons/pencil';

	import type { CollectionName } from '@/pocketbase/collections-models';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import type { RecordEditProps } from './types';

	import { getCollectionManagerContext } from '../collectionManagerContext';

	//

	const props: RecordEditProps<C> = $props();

	const { manager } = $derived(getCollectionManagerContext());

	function openForm() {
		manager.openEditForm(props as RecordEditProps<never>);
	}
</script>

{#if props?.button}
	{@render props.button({
		triggerAttributes: { onclick: openForm },
		icon: Pencil,
		openForm: openForm
	})}
{:else}
	<IconButton variant="outline" icon={Pencil} onclick={openForm} tooltip={m.Edit()} />
{/if}
