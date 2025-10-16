<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SuperForm } from 'sveltekit-superforms';

	import { fromStore } from 'svelte/store';

	import type { CredentialsResponse, WalletsResponse } from '@/pocketbase/types';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Field, FileField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	type LogoFieldConfig = {
		fileFieldName?: 'logo';
	};

	type Props = {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		form: SuperForm<any>;
		recordResponse?: WalletsResponse | CredentialsResponse;
		config?: LogoFieldConfig;
	};

	let { form, recordResponse, config }: Props = $props();
	let formState = fromStore(form.form);

	//

	// Determine which file field name to use
	// Both collections now use 'logo' for file uploads
	const logoFileFieldName = $derived(config?.fileFieldName || 'logo');

	type LogoMode = 'fresh' | 'original' | 'new_file' | 'url' | 'removed';

	const logoMode: LogoMode = $derived.by(() => {
		const currentLogo = formState.current[logoFileFieldName as keyof typeof formState.current];

		// Check for file upload first
		if (currentLogo instanceof File && currentLogo.size > 0) {
			return 'new_file';
		}

		// Check for logo_url field
		if (formState.current.logo_url) {
			return 'url';
		}

		// Check for logo string (URL) - fallback for credentials
		if (typeof currentLogo === 'string' && currentLogo) {
			return 'url';
		}

		if (!recordResponse) {
			return 'fresh';
		} else if (
			currentLogo instanceof File &&
			(recordResponse.logo === currentLogo.name ||
				('logo_file' in recordResponse && recordResponse.logo_file === currentLogo.name))
		) {
			return 'original';
		} else if (!currentLogo && !formState.current.logo_url) {
			return 'removed';
		} else {
			console.warn('Unhandled logo mode', currentLogo);
			return 'fresh';
		}
	});

	const logoPreviewUrl = $derived.by(() => {
		if (logoMode === 'original' && recordResponse) {
			// Both collections use 'logo' field for file uploads
			const logoFile = recordResponse.logo;
			return logoFile ? pb.files.getURL(recordResponse, logoFile) : undefined;
		} else if (logoMode === 'new_file') {
			const currentLogo =
				formState.current[logoFileFieldName as keyof typeof formState.current];
			if (currentLogo instanceof File) {
				return URL.createObjectURL(currentLogo);
			}
		} else if (logoMode === 'url') {
			// Both collections now use logo_url for URL display
			return formState.current.logo_url;
		} else {
			console.warn('Unhandled logo mode', logoMode);
			return undefined;
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
			// Both collections use 'logo' field for file uploads
			if ('logo' in data) {
				data.logo = undefined;
			}
			return data;
		});
	}

	function removeLogoUrl() {
		form.form.update((data) => {
			if ('logo_url' in data) {
				data.logo_url = undefined;
			}
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
			{:else}
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
