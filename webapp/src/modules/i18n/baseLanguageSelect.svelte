<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export type LanguageSelectTriggerSnippetProps = {
		icon: IconComponent;
		text: string;
		language: LanguageData;
	};
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { page } from '$app/state';
	import { Languages } from 'lucide-svelte';

	import type { IconComponent } from '@/components/types';

	import { getLanguagesData, m, type LanguageData } from '.';
	import { getLocale } from './paraglide/runtime';

	type Props = {
		languages: Snippet<[{ languages: LanguageData[] }]>;
		contentClass?: string;
		trigger: Snippet<[LanguageSelectTriggerSnippetProps]>;
	};

	const { trigger, languages: languagesSnippet }: Props = $props();

	const languages = $derived(getLanguagesData(page));
	const currentLanguage = $derived(languages.find((l) => l.tag == getLocale())!);
</script>

{@render trigger({
	icon: Languages,
	text: m.Language(),
	language: currentLanguage
})}

{@render languagesSnippet({ languages })}
