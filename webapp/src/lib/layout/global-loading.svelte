<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { toast } from 'svelte-sonner';

	import { m } from '@/i18n';
	import { getExceptionMessage } from '@/utils/errors';

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

	export async function runWithLoading<T = void>(props: {
		fn: () => Promise<T> | T;
		loadingText?: string;
		successText?: string;
		errorText?: string;
		minDisplayTime?: number;
		showSuccessToast?: boolean;
	}): Promise<T | undefined> {
		const startTime = Date.now();
		const minTime = props.minDisplayTime ?? 1000;
		let spinnerShown = false;

		// Only show spinner if operation takes longer than 1 second
		const spinnerTimeout = setTimeout(() => {
			spinnerShown = true;
			loading.show(props.loadingText);
		}, 300);

		try {
			const result = await props.fn();
			clearTimeout(spinnerTimeout);

			if (spinnerShown) {
				const elapsed = Date.now() - startTime;
				const remaining = minTime - elapsed;
				if (remaining > 0) {
					await new Promise((resolve) => setTimeout(resolve, remaining));
				}
				loading.hide();
			}

			const showToast = props.showSuccessToast ?? true;
			if (showToast) {
				toast.success(props.successText ?? m.Success());
			}
			return result;
		} catch (e) {
			clearTimeout(spinnerTimeout);
			loadingState.error = props.errorText ?? getExceptionMessage(e);
			return undefined;
		}
	}
</script>

<script lang="ts">
	import { TriangleAlert, XIcon } from '@lucide/svelte';

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
