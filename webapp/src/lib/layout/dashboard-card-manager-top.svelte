<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import { RecordCreate } from '@/collections-components/manager';
	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';

	import BlueButton from './blue-button.svelte';

	//

	type Props = {
		label: string;
		buttonText: string;
		recordCreateOptions?: ComponentProps<typeof RecordCreate>;
	};

	let { label, buttonText, recordCreateOptions = {} }: Props = $props();
</script>

<div class="mb-1! flex items-center justify-between">
	<T class="text-sm font-semibold">{label}</T>

	<RecordCreate {...recordCreateOptions}>
		{#snippet button({ triggerAttributes, icon })}
			<BlueButton {...triggerAttributes}>
				<Icon src={icon} />
				{buttonText}
			</BlueButton>
		{/snippet}
	</RecordCreate>
</div>
