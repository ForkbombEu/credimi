// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createForm } from '@/forms';
import { z } from 'zod';
import { stringifiedObjectSchema } from '$lib/start-checks-form/_utils';
import { zod } from 'sveltekit-superforms/adapters';
import type { SuperForm, TaintedFields } from 'sveltekit-superforms';
import { nanoid } from 'nanoid';
import type { State } from '@/utils/types';
import { fromStore } from 'svelte/store';
import { Record, Array as A, Tuple, pipe } from 'effect';
import type { TestConfigFieldsForm } from '$lib/start-checks-form/test-config-fields-form';
import { isNamedTestConfigField } from '$lib/start-checks-form/test-config-field';

//

type TestConfigJsonFormProps = {
	json: string;
	formDependency?: TestConfigFieldsForm;
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
				id: nanoid(6)
			}
		});

		this.taintedState = fromStore(this.superform.tainted);
	}

	reset() {
		this.superform.reset();
	}

	// Placeholders for visualization

	private getPlaceholdersFromJson(): string[] {
		const placeholderRegex = /\{\{\s*\.(\w+)\s*\}\}/g;
		const matches = this.props.json.matchAll(placeholderRegex);
		return Array.from(matches).map((match) => match[1]);
	}

	placeholdersValues: PlaceholderValues = $derived.by(() => {
		if (!this.props.formDependency) return {};

		const placeholders = this.getPlaceholdersFromJson();
		const { validData } = this.props.formDependency.state;

		return pipe(
			this.props.formDependency.props.fields,
			A.filter(isNamedTestConfigField),
			A.filter((field) => placeholders.includes(field.FieldName)),
			A.map((field) => {
				const key = field.FieldName;
				const validValue = validData[field.CredimiID];
				return Tuple.make(key, {
					valid: Boolean(validValue),
					value: validValue ? getValuePreview(validValue) : ''
				});
			}),
			Record.fromEntries
		);
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
