// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { EditorView, ViewPlugin } from '@codemirror/view';
import type { Extension } from '@codemirror/state';

export interface CopyButtonExtensionOptions {
	enabled?: boolean;
	onCopy?: (content: string) => void;
}

export function copyButtonExtension(options: CopyButtonExtensionOptions = {}): Extension {
	const { enabled = true, onCopy } = options;
	
	if (!enabled) return [];

	return ViewPlugin.fromClass(
		class {
			dom: HTMLElement;
			button: HTMLButtonElement;
			isCopied: boolean = false;
			copyTimeout: number | null = null;

			constructor(public view: EditorView) {
				this.dom = document.createElement('div');
				this.dom.className = 'cm-copy-button-container';
				this.dom.style.cssText = `
					position: absolute;
					top: 8px;
					right: 8px;
					z-index: 10;
				`;

				this.button = document.createElement('button');
				this.button.type = 'button';
				this.button.className = 'cm-copy-button';
				this.button.style.cssText = `
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

				this.updateButtonContent();

				this.button.addEventListener('mouseenter', () => {
					this.button.style.opacity = '1';
				});

				this.button.addEventListener('mouseleave', () => {
					this.button.style.opacity = '0.7';
				});

				this.button.addEventListener('click', (event) => {
					event.preventDefault();
					event.stopPropagation();
					this.copyContent();
				});

				this.dom.appendChild(this.button);
				view.dom.style.position = 'relative';
				view.dom.appendChild(this.dom);
			}

			update() {
				// Update button visibility based on content
				const hasContent = this.view.state.doc.length > 0;
				this.dom.style.display = hasContent ? 'block' : 'none';
			}

			destroy() {
				this.dom.remove();
				if (this.copyTimeout) {
					clearTimeout(this.copyTimeout);
				}
			}

			async copyContent() {
				const content = this.view.state.doc.toString();
				
				try {
					await navigator.clipboard.writeText(content);
					this.isCopied = true;
					this.updateButtonContent();

					if (this.copyTimeout) {
						clearTimeout(this.copyTimeout);
					}

					this.copyTimeout = window.setTimeout(() => {
						this.isCopied = false;
						this.updateButtonContent();
					}, 2000);

					onCopy?.(content);
				} catch (err) {
					console.error('Failed to copy text: ', err);
				}
			}

			updateButtonContent() {
				if (this.isCopied) {
					this.button.innerHTML = `
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d="M20 6 9 17l-5-5"/>
						</svg>
					`;
					this.button.style.color = 'hsl(142 76% 36%)'; // green-600
				} else {
					this.button.innerHTML = `
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<rect width="14" height="14" x="8" y="8" rx="2" ry="2"/>
							<path d="m4 16-2-2v-10c0-1.1.9-2 2-2h10l2 2"/>
						</svg>
					`;
					this.button.style.color = 'hsl(var(--foreground))';
				}
			}
		},
		{
			provide: () => EditorView.baseTheme({
				'.cm-copy-button-container': {
					'pointer-events': 'auto'
				},
				'.cm-copy-button': {
					'&:hover': {
						opacity: '1 !important'
					}
				}
			})
		}
	);
}
