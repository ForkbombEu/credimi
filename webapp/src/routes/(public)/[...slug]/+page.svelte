<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import MarkdownRenderer from './markdown-renderer.svelte';

	let { data } = $props();
</script>

<MarkdownRenderer content={data.content} />
