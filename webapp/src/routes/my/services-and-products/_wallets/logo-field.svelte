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

	// const existingLogoUrl = $derived.by(() => {
	// 	if (
	// 		initialData.logo &&
	// 		typeof initialData.logo === 'string' &&
	// 		walletId &&
	// 		!shouldRemoveExistingLogo
	// 	) {
	// 		return pb.files.getURL({ id: walletId, collectionName: 'wallets' }, initialData.logo);
	// 	}
	// 	return null;
	// });

	// let logoUrlError = $state('');

	// let shouldRemoveExistingLogo = $state(false);

	// function removeLogo() {
	// 	const { form: formDataStore } = editWalletform;
	// 	formDataStore.update((currentData) => ({
	// 		...currentData,
	// 		logo: undefined,
	// 		logo_url: ''
	// 	}));
	// 	shouldRemoveExistingLogo = true;
	// 	logoUrlError = '';
	// }

	// $effect(() => {
	// 	if ($formData.logo instanceof File && $formData.logo.size > 0) {
	// 		shouldRemoveExistingLogo = false;
	// 	}
	// });

	// const logoFilePreview = $derived.by(() => {
	// 	const logo = formState.current.logo;
	// 	if (logo && logo.size > 0) {
	// 		return URL.createObjectURL(formState.current.logo);
	// 	}
	// 	return null;
	// });

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
				<img src={logoPreviewUrl} alt={m.Logo_preview()} class="size-full object-cover" />
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

<!-- 
<div class="space-y-4">
	<div class="text-sm font-medium leading-none">{m.Logo()}</div>
	{#if $formData.logo instanceof File && $formData.logo.size > 0 && !logoUrlError}
		{@render logoPreview(URL.createObjectURL($formData.logo))}
	{:else if existingLogoUrl && !$formData.logo && !logoUrlError}
		{@render logoPreview(existingLogoUrl, m.Invalid_image_URL_error())}
	{:else if $formData.logo_url && !logoUrlError}
		{@render logoPreview($formData.logo_url, m.Invalid_image_URL_error())}
	{/if}

	{#if logoUrlError}
		<Alert variant="destructive">
			<AlertCircle class="h-4 w-4" />
			<AlertDescription>{logoUrlError}</AlertDescription>
		</Alert>
	{/if}

	{#if (!($formData.logo instanceof File && $formData.logo.size > 0) && !existingLogoUrl && !$formData.logo_url) || logoUrlError}
		<div class="space-y-4">
			<div class="space-y-2">
				<FileField
					form={editWalletform}
					name="logo"
					options={{
						label: '',
						placeholder: m.Upload_logo(),
						showFilesList: false
					}}
				/>
			</div>
			<div class="relative">
				<div class="absolute inset-0 flex items-center">
					<span class="border-muted w-full border-t"></span>
				</div>
				<div class="relative flex justify-center text-xs uppercase">
					<span class="bg-background text-muted-foreground px-2">{m.or()}</span>
				</div>
			</div>
			<div class="space-y-2">
				<Field
					form={editWalletform}
					name="logo_url"
					options={{
						type: 'url',
						label: '',
						placeholder: m.Enter_logo_URL()
					}}
				/>
			</div>
		</div>
	{/if}
</div>

{#snippet logoPreview(src: string, errorMessage?: string)}
	<div class="relative mb-2 inline-block">
		<img
			{src}
			alt={m.Logo_preview()}
			class="max-h-32 rounded border"
			onerror={() => {
				if (errorMessage) {
					logoUrlError = errorMessage;
				}
			}}
		/>
		<Button
			size="sm"
			variant="destructive"
			class="absolute -right-2 -top-2 h-6 w-6 rounded-full p-0"
			onclick={removeLogo}
		>
			<X class="h-4 w-4" />
		</Button>
	</div>
{/snippet} -->
