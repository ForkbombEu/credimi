<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	export type ConfirmOptions = {
		title?: string;
		message: string;
		confirmLabel?: string;
		cancelLabel?: string;
		destructive?: boolean;
	};

	type ConfirmView = {
		title: string;
		message: string;
		confirmLabel: string;
		cancelLabel: string;
		destructive: boolean;
	};

	// `pending` is the single source of truth for "is a confirm in flight".
	// The dialog is open iff `pending !== null`. `view` only holds what to render
	// and is intentionally kept around while the close animation plays.
	let pending = $state<((confirmed: boolean) => void) | null>(null);
	let currentView = $state<ConfirmView>({
		title: '',
		message: '',
		confirmLabel: '',
		cancelLabel: '',
		destructive: false
	});

	function settle(confirmed: boolean) {
		const resolve = pending;
		if (!resolve) return;
		pending = null;
		resolve(confirmed);
	}

	export function confirm(options: ConfirmOptions): Promise<boolean> {
		// A new request supersedes any in-flight one: resolve it as cancelled first.
		settle(false);

		currentView = {
			title: options.title ?? m.Warning(),
			message: options.message,
			confirmLabel: options.confirmLabel ?? m.Continue(),
			cancelLabel: options.cancelLabel ?? m.Cancel(),
			destructive: options.destructive ?? false
		};

		return new Promise<boolean>((resolve) => {
			pending = resolve;
		});
	}

	export const confirmDialog = {
		get open() {
			return pending !== null;
		},
		get view() {
			return currentView;
		},
		resolve: settle
	};
</script>

<script lang="ts">
	import Button from '@/components/ui-custom/button.svelte';
	import * as AlertDialog from '@/components/ui/alert-dialog';

	const open = $derived(confirmDialog.open);
	const view = $derived(confirmDialog.view);
</script>

<AlertDialog.Root
	{open}
	onOpenChange={(nextOpen) => {
		if (!nextOpen) confirmDialog.resolve(false);
	}}
>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>{view.title}</AlertDialog.Title>
			<AlertDialog.Description class="whitespace-pre-line">
				{view.message}
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<div class="flex justify-center gap-2">
				<Button variant="outline" onclick={() => confirmDialog.resolve(false)}>
					{view.cancelLabel}
				</Button>
				<Button
					variant={view.destructive ? 'destructive' : 'default'}
					onclick={() => confirmDialog.resolve(true)}
				>
					{view.confirmLabel}
				</Button>
			</div>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
