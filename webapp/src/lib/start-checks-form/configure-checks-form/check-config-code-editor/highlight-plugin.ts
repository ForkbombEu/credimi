// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { NamedConfigField } from '$start-checks-form/types';

import { type Extension, StateEffect, StateEffectType } from '@codemirror/state';
import {
	Decoration,
	EditorView,
	MatchDecorator,
	ViewPlugin,
	ViewUpdate,
	WidgetType,
	type DecorationSet
} from '@codemirror/view';
import { formatJson } from '$start-checks-form/_utils';
import _ from 'lodash';

//

export type PlaceholderData = {
	field: NamedConfigField;
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
			const placeholderData = getPlaceholdersData().find(
				(data) => data.field.field_id === fieldName
			);
			if (!placeholderData) return null;

			const line = view.state.doc.lineAt(pos);
			const indentation = line.text.match(/^\s*/)?.[0].length ?? 0;

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
			update(viewUpdate: ViewUpdate) {
				this.placeholders = placeholderMatcher.updateDeco(viewUpdate, this.placeholders);
				if (hasEffect(viewUpdate, 'updatePlaceholders')) {
					this.placeholders = placeholderMatcher.createDeco(viewUpdate.view);
				}
				if (hasEffect(viewUpdate, 'removePlaceholders')) {
					this.placeholders = Decoration.none;
				}
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
		if (!isValid) return `{{ ${field.field_id} }}`;
		if (field.field_type == 'string') return `"${value}"`;
		else return formatJson(value).replaceAll('\n', '\n' + ' '.repeat(this.indentation));
	}
}

// Utils

type Effects = Record<string, StateEffectType<void>>;

const effects = {
	updatePlaceholders: StateEffect.define<void>(),
	removePlaceholders: StateEffect.define<void>()
} satisfies Effects;

type Effect = keyof typeof effects;

export function dispatchEffect(view: EditorView, key: Effect) {
	view.dispatch({ effects: effects[key].of() });
}

function hasEffect(viewUpdate: ViewUpdate, key: Effect) {
	return viewUpdate.transactions.some((t) => t.effects.some((e) => e.is(effects[key])));
}
