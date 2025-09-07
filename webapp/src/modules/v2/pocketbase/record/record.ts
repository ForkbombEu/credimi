// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { toast } from 'svelte-sonner';
import { zod, type ValidationAdapter } from 'sveltekit-superforms/adapters';

import { m } from '@/i18n';
import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
import { form, pocketbaseCrud, task, type db } from '@/v2';

//

type FormMode = 'create' | 'update';

type BaseFormConfig<C extends db.CollectionName> = {
	collection: C;
	crud?: pocketbaseCrud.Instance<C>;
	onSuccess?: <M extends FormMode>(
		mode: M,
		record: db.CollectionResponses[C]
	) => void | Promise<void>;
};

type UpdateFormState<C extends db.CollectionName> = {
	mode: 'update';
	record: db.CollectionResponses[C];
};

type CreateFormState<C extends db.CollectionName> = {
	mode: 'create';
	initialData: Partial<db.CollectionRecords[C]>;
};

type FormState<C extends db.CollectionName> = UpdateFormState<C> | CreateFormState<C>;

type FormConfig<C extends db.CollectionName> = BaseFormConfig<C> & FormState<C>;

export class Form<
	C extends db.CollectionName,
	FormData extends db.CollectionFormData[C] = db.CollectionFormData[C]
> {
	runner = new task.Runner();
	crud: pocketbaseCrud.Instance<C>;
	form: form.Form<FormData>;
	currentState = $state<FormState<C>>();

	constructor(private readonly config: FormConfig<C>) {
		const { collection, crud, ...state } = config;
		this.currentState = state;
		this.crud = crud ?? new pocketbaseCrud.Instance(collection);
		this.form = new form.Form({
			adapter: zod(
				createCollectionZodSchema(collection)
			) as unknown as ValidationAdapter<FormData>,
			onSubmit: (data) => this.submit(data),
			onError: (error) => {
				toast.error(error.message);
			}
		});
	}

	async submit(data: db.CollectionFormData[C]) {
		const {
			runner,
			crud,
			currentState,
			config: { onSuccess }
		} = this;
		let result: db.CollectionResponses[C];

		if (currentState?.mode == 'create') {
			result = await runner.run(crud.create(data));
			toast.success(m.Record_created_successfully());
		} else if (currentState?.mode == 'update') {
			const { record } = currentState;
			result = await runner.run(crud.update(record.id, data));
			toast.success(m.Record_updated_successfully());
		} else {
			return;
		}
		await onSuccess?.(currentState.mode, result);
	}
}
