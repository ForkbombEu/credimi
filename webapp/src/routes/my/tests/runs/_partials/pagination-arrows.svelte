<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeftIcon, ArrowRightIcon } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import SelectInputAny from '@/components/ui-custom/select-input-any.svelte';
	import * as ButtonGroup from '@/components/ui/button-group';

	import type { PaginationParams } from '.';

	type Props = {
		pagination: PaginationParams;
		onPrevious: () => void;
		onNext: () => void;
		onLimitChange: (limit: number) => void;
	};

	let { pagination, onPrevious, onNext, onLimitChange }: Props = $props();
</script>

<SelectInputAny
	items={[
		{ value: 10, label: '10' },
		{ value: 20, label: '20' },
		{ value: 50, label: '50' }
	]}
	value={pagination.limit}
	onValueChange={(v) => {
		if (!v) return;
		onLimitChange(v);
	}}
/>

<ButtonGroup.Root>
	<Button size="icon" variant="outline" onclick={onPrevious} disabled={pagination.offset === 0}>
		<ArrowLeftIcon />
	</Button>
	<Button size="icon" variant="outline" disabled>
		{pagination.offset ?? 0}
	</Button>
	<Button size="icon" variant="outline" onclick={onNext}>
		<ArrowRightIcon />
	</Button>
</ButtonGroup.Root>
