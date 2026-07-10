<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EditorView } from '@codemirror/view';

	import { json } from '@codemirror/lang-json';
	import { yaml } from '@codemirror/lang-yaml';
	import { dev } from '$app/environment';
	import { dracula } from 'thememirror';

	import CodeMirror from './codeEditor.root.svelte';
	import { copyButtonExtension } from './copyButtonExtension.js';

	//

	type LanguageSupport = ReturnType<typeof json>;
	type Extension = typeof dracula;

	const langs = {
		json,
		yaml
	};

	const themes = {
		dracula
	};

	//

	type Props = {
		minHeight?: number | null;
		maxHeight?: number | null;
		value?: string | null | undefined;
		lang: keyof typeof langs | LanguageSupport;
		theme?: keyof typeof themes | Extension;
		class?: string;
		extensions?: Extension[];
		onChange?: (value: string) => void;
		onReady?: (value: EditorView) => void;
		onBlur?: () => void;
		hideCopyButton?: boolean;
		hidePasteButton?: boolean;
		onCopy?: (content: string) => void;
		onPaste?: (content: string) => void;
	};

	let {
		lang,
		minHeight = 100,
		maxHeight,
		theme = 'dracula',
		class: className = '',
		value = $bindable(),
		extensions = [],
		onChange,
		onReady,
		onBlur = () => {},
		hideCopyButton,
		hidePasteButton,
		onCopy,
		onPaste
	}: Props = $props();

	//

	const languageSupport: LanguageSupport | null = $derived.by(() => {
		if (typeof lang == 'string') {
			if (lang in langs) return langs[lang]();
			else return null;
		} else {
			return lang;
		}
	});

	const themeExtension: Extension | null = $derived.by(() => {
		if (typeof theme == 'string') {
			if (theme in themes) return themes[theme];
			else return null;
		} else {
			return theme;
		}
	});

	const styles = $derived.by(() => {
		const rootStyles: { minHeight: string; maxHeight: string; height?: string } = {
			minHeight: 'none',
			maxHeight: 'none'
		};
		if (minHeight) {
			rootStyles.minHeight = `${minHeight}px`;
		} else {
			rootStyles.height = '100%';
		}
		if (maxHeight) rootStyles.maxHeight = `${maxHeight}px`;
		return {
			'&': rootStyles,
			'.cm-scroller': { overflow: 'auto' as const }
		};
	});

	/* Extensions with copy/paste buttons */
	const allExtensions = $derived.by(() => {
		const baseExtensions = [...extensions];
		const showCopy = !hideCopyButton;
		const showPaste = !hidePasteButton;

		if (showCopy || showPaste) {
			baseExtensions.push(
				copyButtonExtension({
					enabled: true,
					showCopy,
					showPaste,
					onCopy,
					onPaste
				})
			);
		}
		return baseExtensions;
	});

	/* Utils */

	function checkParentFlex(el: HTMLElement) {
		if (!dev) return;

		const svelteWrapperElement = el.parentElement;
		const parent = svelteWrapperElement?.parentElement;
		const grandparent = parent?.parentElement;
		if (!grandparent) return;

		const grandparentStyle = window.getComputedStyle(grandparent);
		const parentStyle = window.getComputedStyle(parent);

		if (grandparentStyle.display === 'flex' && !(parentStyle.minWidth === '0px')) {
			console.warn(
				'Warning: CodeEditor grandparent is a flex container. Make sure to set `min-width: 0` on the parent element to prevent overflow issues.'
			);
		}
	}
</script>

<CodeMirror
	lang={languageSupport}
	theme={themeExtension}
	class="overflow-hidden rounded-lg {className}"
	{styles}
	bind:value
	onchange={(e) => {
		onChange?.(e);
	}}
	onready={(view) => {
		checkParentFlex(view.dom);
		view.contentDOM.onblur = onBlur;
		onReady?.(view);
	}}
	extensions={allExtensions}
/>
