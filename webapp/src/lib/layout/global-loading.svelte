<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';
	import { getExceptionMessage } from '@/utils/errors';
	import { toast } from 'svelte-sonner';

	const loadingState = $state({
		text: undefined as string | undefined,
		error: undefined as string | undefined,
		current: false
	});

	export const loading = {
		show: (text?: string) => {
			loadingState.text = text;
			loadingState.current = true;
		},
		hide: () => {
			loadingState.current = false;
		}
	};

	export async function runWithLoading(props: {
		fn: () => Promise<void> | void;
		loadingText?: string;
		successText?: string;
		errorText?: string;
	}) {
		loading.show(props.loadingText);
		try {
			await props.fn();
			loading.hide();
			toast.success(props.successText ?? m.Success());
		} catch (e) {
			loadingState.error = props.errorText ?? getExceptionMessage(e);
		}
	}
</script>

<script lang="ts">
	import { TriangleAlert, XIcon } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import * as AlertDialog from '@/components/ui/alert-dialog';

	//

	const { text, error, current } = $derived(loadingState);

	$effect(() => {
		if (current == false) {
			loadingState.text = undefined;
			loadingState.error = undefined;
		}
	});
</script>

<!--  Some things are a copy of loadingDialog.svelte -->
<AlertDialog.Root bind:open={loadingState.current}>
	<AlertDialog.Content
		class="flex !min-w-[150px] flex-col items-center justify-center gap-2 rounded-sm"
		tabindex={null}
		escapeKeydownBehavior="ignore"
		interactOutsideBehavior="ignore"
	>
		{#if !error}
			<div class="flex items-center gap-2">
				<Spinner size={20} />
				{#if text}
					<AlertDialog.Description>
						{text}
					</AlertDialog.Description>
				{/if}
			</div>
		{:else}
			<div class="flex flex-col items-center gap-4">
				<div class="flex items-center gap-2">
					<TriangleAlert size={20} />
					<AlertDialog.Description>
						{error}
					</AlertDialog.Description>
				</div>
				<Button variant="outline" onclick={() => loading.hide()}>
					<XIcon size={16} />
					{m.Close()}
				</Button>
			</div>
		{/if}
	</AlertDialog.Content>
</AlertDialog.Root>
