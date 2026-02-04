<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { SubmitButton } from '@/forms';

	const { children } = $props();
</script>

<div class="sticky bottom-0 -mx-6 flex justify-end border-t bg-white/70 px-6 py-2 backdrop-blur-sm">
	<SubmitButton>
		{@render children?.()}
	</SubmitButton>
</div>
