<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { EllipsisIcon, PencilIcon } from '@lucide/svelte';

	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	import type { StepsBuilder } from '../steps-builder.svelte.js';

	type Props = {
		builder: StepsBuilder;
		initialYaml: string;
	};

	let { builder, initialYaml }: Props = $props();
</script>

<DropdownMenu
	items={[
		{
			label: m.edit_manually(),
			icon: PencilIcon,
			onclick: () => builder.enterManualMode(initialYaml)
		}
	]}
>
	{#snippet trigger({ props })}
		<IconButton {...props} icon={EllipsisIcon} size="xs" variant="ghost" />
	{/snippet}
</DropdownMenu>
