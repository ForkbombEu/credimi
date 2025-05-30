// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createForm } from '@/forms';
import { z } from 'zod';
import { stringifiedObjectSchema } from '../types';
import { zod } from 'sveltekit-superforms/adapters';
import type { SuperForm, TaintedFields } from 'sveltekit-superforms';
import { nanoid } from 'nanoid';
import type { Getter, State } from '@/utils/types';
import { fromStore } from 'svelte/store';
import { Record } from 'effect';
import type { TestConfigFieldsFormState } from '../test-config-fields-form/test-config-fields-form.svelte.js';

//

type TestConfigJsonFormProps = {
	json: string;
	formStateDependency: Getter<TestConfigFieldsFormState>;
};

type FormData = {
	json: string;
};

export class TestConfigJsonForm {
	superform: SuperForm<FormData>;

	private taintedState: State<TaintedFields<FormData> | undefined>;
	isTainted = $derived.by(() => this.taintedState.current?.json === true);

	constructor(public readonly props: TestConfigJsonFormProps) {
		this.superform = createForm({
			adapter: zod(z.object({ json: stringifiedObjectSchema })),
			initialData: { json: this.props.json },
			options: {
				id: nanoid(6),
				onUpdated: (event) => {
					console.log(event);
				}
			}
		});

		this.taintedState = fromStore(this.superform.tainted);
	}

	// Placeholders for visualization

	getFieldsListFromJson(): string[] {
		const placeholderRegex = /\{\{\s*\.(\w+)\s*\}\}/g;
		const matches = this.props.json.matchAll(placeholderRegex);
		return Array.from(matches).map((match) => match[1]);
	}

	placeholdersValues: PlaceholderValues = $derived.by(() => {
		const placeholders = this.getFieldsListFromJson();
		const { validData, invalidData } = this.props.formStateDependency();

		console.log('validData');
		console.log($state.snapshot(validData));
		console.log('invalidData');
		console.log($state.snapshot(invalidData));

		const baseValues: PlaceholderValues = {
			...Record.map(validData, (value) => ({
				valid: true,
				value: getValuePreview(value)
			})),
			...Record.map(invalidData, (value) => ({
				valid: false,
				value: getValuePreview(value)
			}))
		};

		return Record.filter(baseValues, (_, id) => placeholders.includes(id));
	});
}

// Utils

type PlaceholderValues = Record<string, { valid: boolean; value: string }>;

function getValuePreview(value: string): string {
	try {
		const parsed = JSON.parse(value);
		return JSON.stringify(parsed);
	} catch {
		return value;
	}
}
