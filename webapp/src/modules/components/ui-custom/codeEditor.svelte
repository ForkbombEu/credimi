<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Extension } from '@codemirror/state';
	import type { EditorView } from '@codemirror/view';

	import { json } from '@codemirror/lang-json';
	// import { yaml } from '@codemirror/lang-yaml';
	import { dev } from '$app/environment';
	import CodeMirror from 'svelte-codemirror-editor';
	import { dracula } from 'thememirror';

	import { copyButtonExtension } from './copyButtonExtension.js';

	//

	type LanguageSupport = ReturnType<typeof json>;

	const langs = {
		json
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
		const baseStyles = {
			'&': { minHeight: 'none', maxHeight: 'none' },
			'.cm-scroller': { overflow: 'auto' }
		};
		if (minHeight) baseStyles['&'].minHeight = `${minHeight}px`;
		if (maxHeight) baseStyles['&'].maxHeight = `${maxHeight}px`;
		return baseStyles;
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
	theme={themeExtension ? [themeExtension] : []}
	class="overflow-hidden rounded-lg {className}"
	{styles}
	bind:value
	on:change={(e) => {
		onChange?.(e.detail);
	}}
	on:ready={(e) => {
		const view = e.detail;
		checkParentFlex(view.dom);
		view.contentDOM.onblur = onBlur;
		onReady?.(view);
	}}
	extensions={allExtensions}
/>
