// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { EditorView, ViewPlugin } from '@codemirror/view';
import type { Extension } from '@codemirror/state';

// Lucide SVG strings for consistent iconography
const LUCIDE_ICONS = {
	copy: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="m4 16-2-2v-10c0-1.1.9-2 2-2h10l2 2"/></svg>`,
	clipboard: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="8" height="4" x="8" y="2" rx="1" ry="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/></svg>`,
	check: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>`
};

export interface CopyButtonExtensionOptions {
	enabled?: boolean;
	showCopy?: boolean;
	showPaste?: boolean;
	onCopy?: (content: string) => void;
	onPaste?: (content: string) => void;
}

export function copyButtonExtension(options: CopyButtonExtensionOptions = {}): Extension {
	const { enabled = true, showCopy = true, showPaste = true, onCopy, onPaste } = options;
	
	if (!enabled) return [];

	return ViewPlugin.fromClass(
		class {
			dom: HTMLElement;
			copyButton?: HTMLButtonElement;
			pasteButton?: HTMLButtonElement;
			isCopied: boolean = false;
			isPasted: boolean = false;
			copyTimeout: number | null = null;
			pasteTimeout: number | null = null;
			showCopy: boolean;
			showPaste: boolean;

			constructor(public view: EditorView) {
				this.showCopy = showCopy;
				this.showPaste = showPaste;
				this.dom = document.createElement('div');
				this.dom.className = 'cm-copy-paste-button-container';
				this.dom.style.cssText = `
					position: absolute;
					top: 8px;
					right: 8px;
					z-index: 10;
					display: flex;
					flex-direction: column;
				`;

				// Create paste button if enabled
				if (showPaste) {
					this.createPasteButton();
				}

				// Create copy button if enabled
				if (showCopy) {
					this.createCopyButton();
				}

				view.dom.style.position = 'relative';
				view.dom.appendChild(this.dom);
			}

			update() {
				// Update button visibility based on content
				const hasContent = this.view.state.doc.length > 0;
				
				// Copy button should only be visible when there's content to copy
				if (this.copyButton) {
					this.copyButton.style.display = hasContent ? 'block' : 'none';
				}
				
				// Paste button should always be visible (you can paste into empty editor)
				if (this.pasteButton) {
					this.pasteButton.style.display = 'block';
				}
				
				// Container should be visible if any button is enabled
				const shouldShowContainer = this.showPaste || (this.showCopy && hasContent);
				this.dom.style.display = shouldShowContainer ? 'block' : 'none';
			}

			destroy() {
				this.dom.remove();
				if (this.copyTimeout) {
					clearTimeout(this.copyTimeout);
				}
				if (this.pasteTimeout) {
					clearTimeout(this.pasteTimeout);
				}
			}

			createCopyButton() {
				this.copyButton = document.createElement('button');
				this.copyButton.type = 'button';
				this.copyButton.className = 'cm-copy-button';
				this.copyButton.title = 'Copy to clipboard';
				this.setButtonStyles(this.copyButton);

				this.updateCopyButtonContent();

				this.copyButton.addEventListener('mouseenter', () => {
					this.copyButton!.style.opacity = '1';
				});

				this.copyButton.addEventListener('mouseleave', () => {
					this.copyButton!.style.opacity = '0.7';
				});

				this.copyButton.addEventListener('click', (event) => {
					event.preventDefault();
					event.stopPropagation();
					this.copyContent();
				});

				this.dom.appendChild(this.copyButton);
			}

			createPasteButton() {
				this.pasteButton = document.createElement('button');
				this.pasteButton.type = 'button';
				this.pasteButton.className = 'cm-paste-button';
				this.pasteButton.title = 'Paste from clipboard';
				this.setButtonStyles(this.pasteButton);
				
				// Add margin bottom to separate from copy button
				this.pasteButton.style.marginBottom = '8px';

				this.updatePasteButtonContent();

				this.pasteButton.addEventListener('mouseenter', () => {
					this.pasteButton!.style.opacity = '1';
				});

				this.pasteButton.addEventListener('mouseleave', () => {
					this.pasteButton!.style.opacity = '0.7';
				});

				this.pasteButton.addEventListener('click', (event) => {
					event.preventDefault();
					event.stopPropagation();
					this.pasteContent();
				});

				this.dom.appendChild(this.pasteButton);
			}

			setButtonStyles(button: HTMLButtonElement) {
				button.style.cssText = `
					background: hsl(var(--background) / 0.8);
					border: 1px solid hsl(var(--border) / 0.5);
					color: hsl(var(--foreground));
					border-radius: 6px;
					padding: 6px;
					width: 32px;
					height: 32px;
					cursor: pointer;
					display: flex;
					align-items: center;
					justify-content: center;
					opacity: 0.7;
					transition: opacity 0.2s;
					backdrop-filter: blur(4px);
				`;
			}

			async copyContent() {
				const content = this.view.state.doc.toString();
				
				try {
					await navigator.clipboard.writeText(content);
					this.isCopied = true;
					this.updateCopyButtonContent();

					if (this.copyTimeout) {
						clearTimeout(this.copyTimeout);
					}

					this.copyTimeout = window.setTimeout(() => {
						this.isCopied = false;
						this.updateCopyButtonContent();
					}, 2000);

					onCopy?.(content);
				} catch (err) {
					console.error('Failed to copy text: ', err);
				}
			}

			async pasteContent() {
				try {
					const content = await navigator.clipboard.readText();
					
					// Replace the entire content of the editor
					this.view.dispatch({
						changes: {
							from: 0,
							to: this.view.state.doc.length,
							insert: content
						}
					});

					this.isPasted = true;
					this.updatePasteButtonContent();

					if (this.pasteTimeout) {
						clearTimeout(this.pasteTimeout);
					}

					this.pasteTimeout = window.setTimeout(() => {
						this.isPasted = false;
						this.updatePasteButtonContent();
					}, 2000);

					onPaste?.(content);
				} catch (err) {
					console.error('Failed to paste text: ', err);
				}
			}

			updateCopyButtonContent() {
				if (!this.copyButton) return;
				
				if (this.isCopied) {
					this.copyButton.innerHTML = LUCIDE_ICONS.check;
					this.copyButton.style.color = 'hsl(142 76% 36%)'; // green-600
				} else {
					this.copyButton.innerHTML = LUCIDE_ICONS.copy;
					this.copyButton.style.color = 'hsl(var(--foreground))';
				}
			}

			updatePasteButtonContent() {
				if (!this.pasteButton) return;
				
				if (this.isPasted) {
					this.pasteButton.innerHTML = LUCIDE_ICONS.check;
					this.pasteButton.style.color = 'hsl(142 76% 36%)'; // green-600
				} else {
					this.pasteButton.innerHTML = LUCIDE_ICONS.clipboard;
					this.pasteButton.style.color = 'hsl(var(--foreground))';
				}
			}
		},
		{
			provide: () => EditorView.baseTheme({
				'.cm-copy-paste-button-container': {
					'pointer-events': 'auto'
				},
				'.cm-copy-button, .cm-paste-button': {
					'&:hover': {
						opacity: '1 !important'
					}
				}
			})
		}
	);
}
