<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SuperForm } from 'sveltekit-superforms';

	import { fromStore } from 'svelte/store';

	import type { WalletsFormData, WalletsResponse } from '@/pocketbase/types';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Field, FileField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	type Props = {
		form: SuperForm<WalletsFormData>;
		walletResponse?: WalletsResponse;
	};

	let { form, walletResponse }: Props = $props();
	let formState = fromStore(form.form);

	//

	type LogoMode = 'original' | 'new_file' | 'url' | 'removed';

	const logoMode: LogoMode = $derived.by(() => {
		const currentLogo = formState.current.logo;
		if (walletResponse?.logo === currentLogo?.name) {
			return 'original';
		} else if (currentLogo instanceof File && currentLogo.size > 0) {
			return 'new_file';
		} else if (formState.current.logo_url) {
			return 'url';
		} else if (!currentLogo && !formState.current.logo_url) {
			return 'removed';
		} else {
			throw new Error('Invalid logo mode');
		}
	});

	const logoPreviewUrl = $derived.by(() => {
		if (logoMode === 'original' && walletResponse) {
			return pb.files.getURL(walletResponse, walletResponse.logo);
		} else if (logoMode === 'new_file' && formState.current.logo) {
			return URL.createObjectURL(formState.current.logo);
		} else if (logoMode === 'url') {
			return formState.current.logo_url;
		} else if (logoMode === 'removed') {
			return undefined;
		} else {
			throw new Error('Invalid logo mode');
		}
	});

	//

	function removeLogo() {
		if (logoMode === 'original' || logoMode === 'new_file') {
			removeLogoFile();
		} else if (logoMode === 'url') {
			removeLogoUrl();
		}
	}

	function removeLogoFile() {
		form.form.update((data) => {
			data.logo = undefined;
			return data;
		});
	}

	function removeLogoUrl() {
		form.form.update((data) => {
			data.logo_url = undefined;
			return data;
		});
	}
</script>

<div class="flex gap-4">
	<div class="grow">
		<FileField
			{form}
			name="logo"
			options={{
				label: m.Upload_logo(),
				placeholder: m.Upload_logo(),
				showFilesList: false
			}}
		/>

		{@render or()}

		<div class="pt-2">
			<Field
				{form}
				name="logo_url"
				options={{
					type: 'url',
					hideLabel: true,
					placeholder: m.Enter_logo_URL()
				}}
			/>
		</div>
	</div>

	<div class="relative mt-8">
		<div class="size-28 overflow-hidden rounded-md border">
			{#if logoPreviewUrl}
				<img
					src={logoPreviewUrl}
					alt={m.Logo_preview()}
					class="size-full object-cover text-center text-xs"
				/>
			{:else if logoMode === 'removed'}
				<div class="bg-muted flex size-full items-center justify-center p-2">
					<T class="text-muted-foreground text-sm">Logo preview</T>
				</div>
			{/if}
		</div>
		{#if logoPreviewUrl}
			<IconButton
				size="sm"
				variant="destructive"
				class="absolute -right-2 -top-2 h-6 w-6 rounded-full p-0"
				onclick={removeLogo}
			/>
		{/if}
	</div>
</div>

{#snippet or()}
	<div class="relative">
		<div class="absolute inset-0 flex items-center">
			<span class="border-muted w-full border-t"></span>
		</div>
		<div class="relative flex justify-center text-xs uppercase">
			<span class="bg-background text-muted-foreground px-2">{m.or()}</span>
		</div>
	</div>
{/snippet}
