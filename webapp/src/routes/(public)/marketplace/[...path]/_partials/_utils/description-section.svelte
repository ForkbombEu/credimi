<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { marked } from 'marked';

	import RenderHTML from '@/components/ui-custom/renderHTML.svelte';

	import PageSection from './page-section.svelte';
	import { sections as s } from './sections';

	//

	type Props = {
		description?: string;
	};

	let { description }: Props = $props();

	const content = $derived(marked.parse(description ?? '', { async: false }));

	const isEmpty = $derived.by(() => {
		const parser = new DOMParser();
		const doc = parser.parseFromString(content, 'text/html');
		const textContent = doc.body.textContent || '';
		return textContent.trim() === '';
	});
</script>

<PageSection indexItem={s.description} empty={isEmpty}>
	<div class="prose">
		<RenderHTML {content} />
	</div>
</PageSection>
