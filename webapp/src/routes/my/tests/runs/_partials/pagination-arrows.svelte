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

	//

	type Props = {
		pagination: PaginationParams;
		onPrevious: () => void;
		onNext: () => void;
		onLimitChange: (limit: number) => void;
		currentItemCount: number;
	};

	let { pagination, onPrevious, onNext, onLimitChange, currentItemCount }: Props = $props();
</script>

<SelectInputAny
	items={[10, 20, 50].map((value) => ({ value, label: String(value) }))}
	value={pagination.limit}
	onValueChange={(v) => {
		if (!v) return;
		onLimitChange(v);
	}}
/>

<ButtonGroup.Root>
	<Button
		size="icon"
		variant="outline"
		onclick={onPrevious}
		disabled={pagination.offset === 0 || pagination.offset === undefined}
	>
		<ArrowLeftIcon />
	</Button>
	<Button size="icon" variant="outline" disabled>
		{(pagination.offset ?? 0) + 1}
	</Button>
	<Button
		size="icon"
		variant="outline"
		onclick={onNext}
		disabled={currentItemCount < (pagination.limit ?? 0)}
	>
		<ArrowRightIcon />
	</Button>
</ButtonGroup.Root>
