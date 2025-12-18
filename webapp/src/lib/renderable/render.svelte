<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Renderable } from './types';

	let { item }: { item: Renderable } = $props();
</script>

<item.Component self={item} />
