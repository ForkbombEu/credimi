// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	Decoration,
	EditorView,
	MatchDecorator,
	ViewPlugin,
	ViewUpdate,
	WidgetType,
	type DecorationSet
} from '@codemirror/view';
import { type Extension, StateEffect } from '@codemirror/state';
import _ from 'lodash';
import type { NamedTestConfigField } from '../test-config-field';
import { formatJson } from '../_utils';

//

export type PlaceholderData = {
	field: NamedTestConfigField;
	isValid: boolean;
	value: string;
};

type DisplayPlaceholderDataSettings = {
	placeholdersRegex: RegExp;
	getPlaceholdersData: () => PlaceholderData[];
};

export function displayPlaceholderData(settings: DisplayPlaceholderDataSettings): Extension {
	const { placeholdersRegex, getPlaceholdersData } = settings;

	const placeholderMatcher = new MatchDecorator({
		regexp: placeholdersRegex,
		decoration: (match, view, pos) => {
			const fieldName = match[1];
			const line = view.state.doc.lineAt(pos);
			const indentation = line.text.match(/^\s*/)?.[0].length ?? 0;

			const placeholderData = getPlaceholdersData().find(
				(data) => data.field.FieldName === fieldName
			);
			if (!placeholderData) return null;

			return Decoration.replace({
				widget: new PlaceholderWidget(placeholderData, indentation)
			});
		}
	});

	const plugin = ViewPlugin.fromClass(
		class {
			placeholders: DecorationSet;
			constructor(view: EditorView) {
				this.placeholders = placeholderMatcher.createDeco(view);
			}
			update(update: ViewUpdate) {
				this.placeholders = placeholderMatcher.createDeco(update.view);
			}
		},
		{
			decorations: (instance) => instance.placeholders,
			provide: (plugin) =>
				EditorView.atomicRanges.of((view) => {
					return view.plugin(plugin)?.placeholders || Decoration.none;
				})
		}
	);

	return plugin.extension;
}

class PlaceholderWidget extends WidgetType {
	constructor(
		private data: PlaceholderData,
		private indentation: number
	) {
		super();
	}

	eq(other: PlaceholderWidget) {
		return _.isEqual(this.data, other.data);
	}

	toDOM() {
		const span = document.createElement('span');
		span.textContent = this.getTextContent();

		span.className = 'rounded px-1 !leading-[0.7]';
		if (this.data.isValid) {
			span.className += ' bg-green-500/80';
		} else {
			span.className += ' bg-red-500/80';
		}

		return span;
	}

	ignoreEvent() {
		return false;
	}

	getTextContent() {
		const { field, isValid, value } = this.data;
		if (!isValid) return `{{ ${field.FieldName} }}`;
		if (field.Type == 'string') return `"${value}"`;
		else return formatJson(value).replaceAll('\n', '\n' + ' '.repeat(this.indentation));
	}
}

//

const refreshEffect = StateEffect.define<void>();

export function refreshEditorView(view: EditorView) {
	view.dispatch({ effects: refreshEffect.of() });
}
