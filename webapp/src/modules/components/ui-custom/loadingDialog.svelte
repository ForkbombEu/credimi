<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import * as AlertDialog from '@/components/ui/alert-dialog';

	import Spinner from './spinner.svelte';

	//

	interface Props {
		loading?: boolean;
		contentClass?: string;
		children?: Snippet;
		bottom?: Snippet;
		top?: Snippet;
	}

	let { loading = $bindable(false), children, contentClass, bottom, top }: Props = $props();
</script>

<AlertDialog.Root bind:open={loading}>
	<AlertDialog.Content
		class={[
			'flex !min-w-[150px] flex-col items-center justify-center gap-2 rounded-sm',
			contentClass
		]}
		tabindex={null}
		escapeKeydownBehavior="ignore"
		interactOutsideBehavior="ignore"
	>
		{@render top?.()}

		<Spinner />

		{#if children}
			<AlertDialog.Description>
				{@render children()}
			</AlertDialog.Description>
		{/if}

		{@render bottom?.()}
	</AlertDialog.Content>
</AlertDialog.Root>
