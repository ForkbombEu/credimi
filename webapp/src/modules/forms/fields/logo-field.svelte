<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import { UploadIcon } from '@lucide/svelte';
	import { fromStore } from 'svelte/store';
	import { fieldProxy, type FormPathLeaves, type SuperForm } from 'sveltekit-superforms';

	import type { GenericRecord } from '@/utils/types';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { readFileAsDataURL } from '@/utils/files';

	import FileField from './fileField.svelte';

	//

	type Props = {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, File>;
		initialPreviewUrl?: string;
		label?: string;
	};

	let { form, name, initialPreviewUrl, label = m.Logo() }: Props = $props();

	//

	let previewUrl = $state(initialPreviewUrl);
	const value = fromStore(fieldProxy(form, name));
	const initialValue = value.current;

	$effect(() => {
		const logo = value.current as File | undefined | null;
		if (logo) {
			if (logo.size > 0) {
				readFileAsDataURL(logo).then((dataURL) => {
					previewUrl = dataURL;
				});
			} else {
				previewUrl = initialPreviewUrl;
			}
		} else {
			previewUrl = undefined;
		}
	});

	function removeLogo() {
		// @ts-expect-error - undefined is a valid value for a file field
		value.current = undefined;
	}

	const canResetLogo = $derived(initialValue && initialValue !== value.current);

	function resetLogo() {
		if (canResetLogo && initialValue) {
			value.current = initialValue;
		}
	}
</script>

<div class="flex items-start gap-4">
	<div class="grow">
		<FileField
			{form}
			variant="outline"
			{name}
			options={{ label, labelRight, showFilesList: false }}
		>
			<UploadIcon />
			<T>{m.Upload_logo()}</T>
		</FileField>

		{#snippet labelRight()}
			<button
				onclick={(e) => {
					e.preventDefault();
					resetLogo();
				}}
				disabled={!canResetLogo}
				class={[
					'text-primary cursor-pointer text-sm underline underline-offset-2',
					{ invisible: !canResetLogo }
				]}
			>
				{m.Reset()}
			</button>
		{/snippet}
	</div>

	<div class="relative pt-1">
		<Avatar
			src={previewUrl}
			alt={m.Logo_preview()}
			class="size-16 rounded-md border bg-slate-50"
		/>
		{#if value.current}
			<IconButton
				size="sm"
				variant="destructive"
				class="absolute -right-2 -top-1 h-6 w-6 rounded-full p-0 hover:bg-red-600"
				onclick={removeLogo}
			/>
		{/if}
	</div>
</div>
