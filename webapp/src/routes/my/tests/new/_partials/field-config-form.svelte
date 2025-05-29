<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CodeEditorField } from '@/forms/fields';
	import { PlaceholderHighlightCodeEditorField } from '@/forms/fields';
	import { createForm, Form } from '@/forms';
	import { zod } from 'sveltekit-superforms/adapters';
	import {
		createInitialDataFromFields,
		createTestVariablesFormSchema,
		type TestInput
	} from './logic';
	import { Record as R, Record, pipe, Array as A } from 'effect';
	import { Store, watch } from 'runed';
	import FieldConfigToFormField from './field-config-to-form-field.svelte';
	import Label from '@/components/ui/label/label.svelte';
	import { Pencil, Info, Undo, Eye } from 'lucide-svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Button } from '@/components/ui/button';
	import Separator from '@/components/ui/separator/separator.svelte';
	import { nanoid } from 'nanoid';
	import * as Popover from '@/components/ui/popover/index.js';
	import { stringifiedObjectSchema, type SpecificFieldConfig } from './tests-configs-form/types';

	//

	type UpdateFunction = (testInput: TestInput) => void;

	type Props = {
		fields: SpecificFieldConfig[];
		jsonConfig: Record<string, unknown>;
		defaultFieldsIds?: string[];
		defaultValues?: Record<string, unknown>;
		onValidUpdate?: UpdateFunction;
		onInvalidUpdate?: () => void;
	};

	const {
		fields,
		defaultValues = {},
		defaultFieldsIds = [],
		onValidUpdate,
		onInvalidUpdate,
		jsonConfig = {}
	}: Props = $props();

	/* Form creation */

	const JSON_CONFIG_KEY = 'jsonConfig';
	const jsonConfigString = JSON.stringify(jsonConfig, null, 4);

	const form = createForm({
		adapter: zod(
			createTestVariablesFormSchema(fields).extend({
				jsonConfig: stringifiedObjectSchema.optional()
			})
		),
		initialData: {
			[JSON_CONFIG_KEY]: jsonConfigString,
			...createInitialDataFromFields(fields, defaultFieldsIds)
		},
		options: {
			id: nanoid(6)
		}
	});

	const { form: formData } = form;

	/* Fields organization */

	const specificFields = $derived(fields.filter((f) => !defaultFieldsIds.includes(f.CredimiID)));

	let overriddenFieldsIds = $state<string[]>([]);

	const overriddenFields = $derived(
		fields.filter((f) => overriddenFieldsIds.includes(f.CredimiID))
	);

	function resetOverride(id: string) {
		overriddenFieldsIds = overriddenFieldsIds.filter((f) => f !== id);
		$formData = { ...$formData, [id]: defaultValues[id] };
	}

	const defaultFields = $derived(
		pipe(fields, A.difference(specificFields), A.difference(overriddenFields))
	);

	/* Update form data when default values change */

	watch(
		() => defaultValues,
		() => {
			const notOverridden = pipe(
				defaultValues,
				// Only fields that are in the fields array
				R.filter((_, key) => fields.map((f) => f.CredimiID).includes(key)),
				// Not overridden
				R.filter((_, key) => !overriddenFieldsIds.includes(key))
			);
			$formData = { ...$formData, ...notOverridden };
		}
	);

	/* Disable fields when jsonConfig is manually edited */

	const { tainted } = form;

	const taintedState = new Store(tainted);
	const isJsonConfigTainted = $derived(Boolean(taintedState.current?.jsonConfig));

	function resetJsonConfig() {
		$formData = { ...$formData, jsonConfig: jsonConfigString };
	}

	/* Trigger onValidUpdate */

	const { validateForm, validate, errors } = form;
	const formState = new Store(formData); // Readonly
	const errorsState = new Store(errors); // Track errors reactively

	// Remove the field-specific watchers that might cause infinite loops

	watch(
		() => formState.current,
		() => {
			if (isJsonConfigTainted) {
				const jsonConfigString = formState.current[JSON_CONFIG_KEY] as string;

				validate(JSON_CONFIG_KEY, { update: false, value: jsonConfigString }).then(
					(errors) => {
						if (Boolean(errors?.length)) onInvalidUpdate?.();
						else
							onValidUpdate?.({
								format: 'json',
								data: jsonConfigString
							});
					}
				);
			} else {
				const testInput: TestInput = {
					format: 'variables',
					data: pipe(
						formState.current,
						R.remove(JSON_CONFIG_KEY),
						R.map((value, credimiId) => {
							const field = fields.find((f) => f.CredimiID === credimiId);
							if (!field) throw new Error(`Field ${credimiId} not found`);
							return {
								type: field.Type,
								value: value,
								fieldName: field.FieldName
							};
						})
					)
				};
				validateForm({ update: false }).then((result) => {
					if (result.valid) onValidUpdate?.(testInput);
					else onInvalidUpdate?.();
				});
			}
		}
	);

	// Create a reactive store for the field validation state using a simpler approach
	let fieldValidationStateValue =
		$state<Record<string, { valid: boolean; value: string }>>(getFieldValidationState());

	// Update validation state when form data or errors change
	$effect(() => {
		// Access these to track changes
		formState.current;
		errorsState.current;

		// Update validation state
		fieldValidationStateValue = getFieldValidationState();
	});

	/* Utils */

	function previewValue(value: unknown, type: SpecificFieldConfig['Type']): string {
		const NULL_VALUE = '<null>';
		if (!value) return NULL_VALUE;
		if (type === 'string') return value as string;
		if (type === 'object') return JSON.stringify(JSON.parse(value as string), null, 4);
		return NULL_VALUE;
	}

	function getPlaceholdersFromJson(jsonString: string): string[] {
		const placeholderRegex = /\{\{\s*\.(\w+)\s*\}\}/g;
		const placeholders = new Set<string>();
		let match;

		while ((match = placeholderRegex.exec(jsonString)) !== null) {
			placeholders.add(match[1]);
		}

		return [...placeholders];
	}

	function getFieldValidationState() {
		const validationState: Record<string, { valid: boolean; value: string }> = {};
		const jsonString = formState.current[JSON_CONFIG_KEY] as string;

		// Get all placeholders from the JSON
		const jsonPlaceholders = getPlaceholdersFromJson(jsonString);

		// Process fields from formData
		for (const field of fields) {
			const fieldId = field.CredimiID;
			const value = formState.current[fieldId] as string;
			if (!value) continue;

			// Check for errors using the Superforms errors object
			const fieldErrors = errorsState.current?.[fieldId];
			const hasErrors = Array.isArray(fieldErrors) && fieldErrors.length > 0;
			const isValid = !hasErrors;

			validationState[field.FieldName] = {
				valid: isValid,
				value: field.Type === 'object' ? truncateObjectPreview(value) : value
			};
		}

		// Add placeholders without values to the validation state
		for (const placeholder of jsonPlaceholders) {
			if (!validationState[placeholder]) {
				validationState[placeholder] = {
					valid: false,
					value: '' // Empty value will trigger default placeholder behavior
				};
			}
		}

		return validationState;
	}

	function truncateObjectPreview(jsonValue: string): string {
		try {
			// For JSON objects, we want to preserve the actual stringified value
			// but clean it up to remove extra whitespace
			const parsed = JSON.parse(jsonValue);
			return JSON.stringify(parsed);
		} catch (e) {
			// If we can't parse it, just return it as is
			return jsonValue || '';
		}
	}
</script>

<Form {form} hide={['submit_button']} hideRequiredIndicator class="flex flex-col gap-8 md:flex-row">
	<!--  -->
	<div class="flex min-w-0 shrink-0 grow basis-1 flex-col">
		{#if isJsonConfigTainted}
			<CodeEditorField
				{form}
				name={JSON_CONFIG_KEY}
				options={{
					lang: 'json',
					label: 'JSON configuration',
					class: 'self-stretch'
				}}
			/>
		{:else}
			<PlaceholderHighlightCodeEditorField
				{form}
				name={JSON_CONFIG_KEY}
				fieldValues={fieldValidationStateValue}
				options={{
					lang: 'json',
					label: 'JSON configuration',
					class: 'self-stretch'
				}}
			/>
		{/if}
	</div>

	<div class="min-w-0 shrink-0 grow basis-1">
		<div class="mb-8 space-y-2">
			<Label>Fields</Label>
			<Separator />
		</div>
		{#if isJsonConfigTainted}
			<div class="text-muted-foreground text-sm">
				<Alert variant="info" icon={Info}>
					{#snippet content({ Title, Description })}
						<Title class="font-bold">Info</Title>
						<Description class="mb-2">
							JSON configuration is manually edited. Fields are disabled.
						</Description>

						<Button
							variant="outline"
							onclick={(e) => {
								e.preventDefault();
								resetJsonConfig();
							}}
						>
							Reset JSON and use fields
						</Button>
					{/snippet}
				</Alert>
			</div>
		{:else}
			<div class="space-y-8">
				{#each specificFields as config}
					<FieldConfigToFormField {config} {form} />
				{/each}

				{#each overriddenFields as config}
					<div class="relative">
						<div class="absolute right-0 top-0">
							<button
								class="text-primary flex items-center gap-2 text-sm underline hover:no-underline"
								onclick={(e) => {
									e.preventDefault(); // Important to prevent form submission
									resetOverride(config.CredimiID);
								}}
							>
								<Icon src={Undo} size={14} />
								Reset to default
							</button>
						</div>
						<FieldConfigToFormField {config} {form} />
					</div>
				{/each}

				{#if defaultFields.length}
					<div class="space-y-2">
						<Label>Default fields</Label>
						<ul class="space-y-1">
							{#each defaultFields as { CredimiID, LabelKey, Type }}
								{@const value = defaultValues[CredimiID]}
								{@const valuePreview = previewValue(value, Type)}

								<li class="flex items-center gap-2">
									<span class="font-mono text-sm">{LabelKey}</span>

									<Popover.Root>
										<Popover.Trigger
											class="rounded-md p-1 hover:cursor-pointer hover:bg-gray-200"
										>
											<Eye size={14} />
										</Popover.Trigger>
										<Popover.Content class="dark overflow-auto">
											<pre class="text-xs">{valuePreview}</pre>
										</Popover.Content>
									</Popover.Root>

									<button
										class="rounded-md p-1 hover:cursor-pointer hover:bg-gray-200"
										onclick={(e) => {
											e.preventDefault(); // Important to prevent form submission
											overriddenFieldsIds.push(CredimiID);
										}}
									>
										<Pencil size={14} />
									</button>
								</li>
							{/each}
						</ul>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</Form>
