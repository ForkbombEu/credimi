<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { ComponentProps, Snippet } from 'svelte';
	import type { Writable } from 'svelte/store';
	import type { FormPath, SuperForm } from 'sveltekit-superforms';

	import { UploadIcon } from 'lucide-svelte';
	import { fieldProxy } from 'sveltekit-superforms';

	import type { GenericRecord } from '@/utils/types';

	import FileManager from '@/components/ui-custom/fileManager.svelte';
	import { Button } from '@/components/ui/button';
	import * as Form from '@/components/ui/form';
	import Input from '@/components/ui/input/input.svelte';

	import type { FieldOptions } from './types';

	import { createFilesValidator } from './fileField';
	import FieldWrapper from './parts/fieldWrapper.svelte';

	//

	type Props = {
		form: SuperForm<Data>;
		name: FormPath<Data>;
		variant?: ComponentProps<typeof Button>['variant'];
		class?: string;
		options?: Partial<
			FieldOptions &
				Omit<ComponentProps<typeof Input>, 'type' | 'value'> & { showFilesList?: boolean }
		>;
		children?: Snippet;
	};

	const {
		form,
		name,
		class: className,
		variant = 'outline',
		options = {},
		children
	}: Props = $props();

	const multiple = $derived(options.multiple ?? false);
	const valueProxy = $derived(fieldProxy(form, name) as Writable<File | File[]>);

	const validator = $derived(
		createFilesValidator(form as SuperForm<GenericRecord>, name, multiple)
	);

	let fileInput: HTMLInputElement;
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#snippet children({ props })}
			<FileManager
				bind:data={$valueProxy}
				{validator}
				{multiple}
				showFilesList={options.showFilesList}
			>
				{#snippet children({ addFiles })}
					<Button
						type="button"
						{variant}
						onclick={() => fileInput.click()}
						class={['w-full', className]}
					>
						{#if children}
							{@render children({ addFiles })}
						{:else}
							<UploadIcon />{options.placeholder}
						{/if}
					</Button>
					<input
						bind:this={fileInput}
						{...props}
						type="file"
						{multiple}
						accept={options.accept}
						class="hidden"
						onchange={(e) => {
							const fileList = e.currentTarget.files;
							if (fileList) addFiles([...fileList]);
							e.currentTarget.value = '';
						}}
					/>
					<!-- e.currentTarget.value = '' is needed to clear the file input -->
				{/snippet}
			</FileManager>
		{/snippet}
	</FieldWrapper>
</Form.Field>
