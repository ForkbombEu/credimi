// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Decoration, EditorView, ViewPlugin, WidgetType } from '@codemirror/view';
import { StateEffect, StateField } from '@codemirror/state';
import { onDestroy } from 'svelte';
import _ from 'lodash';

type PlaceholderState = {
	isValid: boolean;
	value: string;
};

class PlaceholderStateWidget extends WidgetType {
	state: PlaceholderState;

	constructor(state: PlaceholderState) {
		super();
		this.state = state;
	}

	eq(other: PlaceholderStateWidget) {
		return _.isEqual(this.state, other.state);
	}

	toDOM() {
		const span = document.createElement('span');
		span.textContent = this.state.value;

		if (this.state.isValid) {
			span.className = 'bg-green-500/80 rounded px-1';
		} else {
			span.className = 'bg-red-500/80 rounded px-1';
		}

		return span;
	}

	ignoreEvent() {
		return false;
	}
}

// StateEffect for updating field values
const updatePlaceholdersStatesEffect =
	StateEffect.define<Record<string, { valid: boolean; value: string }>>();

// StateField to track field values in editor state
const fieldValuesField = StateField.define<Record<string, { valid: boolean; value: string }>>({
	create() {
		return {};
	},
	update(values, tr) {
		for (let e of tr.effects) {
			if (e.is(updatePlaceholdersStatesEffect)) {
				values = e.value;
			}
		}
		return values;
	}
});

// ViewPlugin that creates decorations based on placeholders
const placeholderPlugin = ViewPlugin.fromClass(
	class {
		decorations;
		fieldValues: Record<string, { valid: boolean; value: string }>;

		constructor(view: EditorView) {
			this.fieldValues = view.state.field(fieldValuesField);
			this.decorations = this.createDecorations(view);
		}

		update(update: { view: EditorView; docChanged: boolean; viewportChanged: boolean }) {
			if (
				update.docChanged ||
				update.viewportChanged ||
				update.view.state.field(fieldValuesField) !== this.fieldValues
			) {
				this.fieldValues = update.view.state.field(fieldValuesField);
				this.decorations = this.createDecorations(update.view);
			}
		}

		createDecorations(view: EditorView) {
			const decorations = [];
			const doc = view.state.doc;
			// Updated regex to match both JSON and regular placeholders
			const placeholderRegex = /"?\{\{\s*\.(\w+)\s*\}\}"?/g;

			// Find all placeholders in visible ranges
			for (let { from, to } of view.visibleRanges) {
				const text = doc.sliceString(from, to);
				let match;
				while ((match = placeholderRegex.exec(text)) !== null) {
					const fieldName = match[1];
					const matchFrom = from + match.index;
					const matchTo = matchFrom + match[0].length;
					const originalPlaceholder = match[0];
					const isJsonPlaceholder =
						originalPlaceholder.startsWith('"{{') &&
						originalPlaceholder.endsWith('}}"');

					const fieldInfo = this.fieldValues[fieldName];

					if (fieldInfo && fieldInfo.value) {
						let displayValue = fieldInfo.value;

						if (isJsonPlaceholder) {
							try {
								// If it's valid JSON, we'll use it without quotes
								JSON.parse(fieldInfo.value);
								displayValue = fieldInfo.value.trim();
							} catch (e) {
								// If not valid JSON, treat it as a regular string with quotes
								displayValue = `"${fieldInfo.value}"`;
							}
						} else {
							// For non-JSON placeholders, always add quotes
							displayValue = `"${fieldInfo.value}"`;
						}

						// Replace the placeholder with the actual value
						decorations.push(
							Decoration.replace({
								widget: new PlaceholderStateWidget(displayValue, fieldInfo.valid)
							}).range(matchFrom, matchTo)
						);
					} else {
						// Keep the original placeholder but style it based on status
						decorations.push(
							Decoration.mark({
								class: 'bg-red-500/80 rounded px-1'
							}).range(matchFrom, matchTo)
						);
					}
				}
			}

			return Decoration.set(decorations, true);
		}
	},
	{
		decorations: (instance) => instance.decorations
	}
);

// Custom function to handle editor initialization
function handleReady(view: EditorView) {
	// Update field values when they change
	view.dispatch({
		effects: updatePlaceholdersStatesEffect.of(fieldValues || {})
	});

	// Pass the view to the original onReady if provided
	if (onReady) {
		onReady(view);
	}
}

let editorView: EditorView | null = null;

// Combine our extensions with any passed in
const combinedExtensions = [fieldValuesField, placeholderPlugin, ...extensions];

// Watch for changes in fieldValues prop with debouncing
let previousFieldValuesJSON = '';
let updateTimeout: ReturnType<typeof setTimeout>;

$effect(() => {
	if (!editorView) return;

	// Convert to JSON string to do deep comparison
	const currentFieldValuesJSON = JSON.stringify(fieldValues);

	// Only update if values actually changed
	if (currentFieldValuesJSON !== previousFieldValuesJSON) {
		previousFieldValuesJSON = currentFieldValuesJSON;

		// Debounce updates
		clearTimeout(updateTimeout);
		updateTimeout = setTimeout(() => {
			if (editorView) {
				editorView.dispatch({
					effects: updatePlaceholdersStatesEffect.of(fieldValues)
				});
			}
		}, 50);
	}
});

// Clean up timeout on component unmount
onDestroy(() => {
	clearTimeout(updateTimeout);
});
