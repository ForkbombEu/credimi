<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as AlertDialog from '@/components/ui/alert-dialog';
	import Spinner from './spinner.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		loading?: boolean;
		children?: Snippet;
	}

	let { loading = $bindable(false), children }: Props = $props();
</script>

<AlertDialog.Root bind:open={loading}>
	<AlertDialog.Content
		class="flex !w-1/5 !min-w-[150px] flex-col items-center justify-center gap-2 rounded-sm"
		tabindex={null}
		escapeKeydownBehavior="ignore"
		interactOutsideBehavior="ignore"
	>
		<Spinner />
		<AlertDialog.Description>
			{@render children?.()}
		</AlertDialog.Description>
	</AlertDialog.Content>
</AlertDialog.Root>
