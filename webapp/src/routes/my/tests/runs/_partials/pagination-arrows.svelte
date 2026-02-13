<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeftIcon, ArrowRightIcon } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import * as ButtonGroup from '@/components/ui/button-group';

	import type { PaginationParams } from '.';

	type Props = {
		pagination: PaginationParams;
		onPrevious: () => void;
		onNext: () => void;
	};

	let { pagination, onPrevious, onNext }: Props = $props();
</script>

<ButtonGroup.Root>
	<Button size="icon" variant="outline" onclick={onPrevious} disabled={pagination.offset === 0}>
		<ArrowLeftIcon />
	</Button>
	<Button size="icon" variant="outline" disabled>
		{pagination.offset}
	</Button>
	<Button size="icon" variant="outline" onclick={onNext}>
		<ArrowRightIcon />
	</Button>
</ButtonGroup.Root>
