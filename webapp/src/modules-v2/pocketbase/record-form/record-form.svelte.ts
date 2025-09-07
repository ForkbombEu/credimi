// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { form, pocketbase as pb, pocketbaseCrud, task, ui, type db } from '#';
import { zod, type ValidationAdapter } from 'sveltekit-superforms/adapters';

import { m } from '@/i18n';
import { createCollectionZodSchema } from '@/pocketbase/zod-schema';

import { recordToFormData } from './functions';

//

export type Mode = 'create' | 'update';

type BaseConfig<C extends db.CollectionName> = {
	collection: C;
	crud?: pocketbaseCrud.Instance<C>;
	onSuccess?: <M extends Mode>(
		mode: M,
		record: db.CollectionResponses[C]
	) => void | Promise<void>;
};

type UpdateState<C extends db.CollectionName> = {
	mode: 'update';
	record: db.CollectionResponses[C];
};

type CreateState<C extends db.CollectionName> = {
	mode: 'create';
	initialData: Partial<db.CollectionRecords[C]>;
};

type State<C extends db.CollectionName> = UpdateState<C> | CreateState<C>;

type Config<C extends db.CollectionName> = BaseConfig<C> & State<C>;

export class Instance<
	C extends db.CollectionName,
	FormData extends db.CollectionFormData[C] = db.CollectionFormData[C]
> {
	runner = new task.Runner();
	crud: pocketbaseCrud.Instance<C>;
	form: form.Instance<FormData>;

	private currentState = $state<State<C>>();
	get currentMode() {
		return this.currentState?.mode;
	}

	constructor(private readonly config: Config<C>) {
		const { collection, crud, ...state } = config;
		this.currentState = state;
		this.crud = crud ?? new pocketbaseCrud.Instance(collection);

		const adapter = zod(
			createCollectionZodSchema(collection)
		) as unknown as ValidationAdapter<FormData>;

		this.form = new form.Instance({
			adapter,
			onSubmit: (data) => this.submit(data),
			onError: (error) => {
				ui.toast.error(error.message);
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
			ui.toast.success(m.Record_created_successfully());
		} else if (currentState?.mode == 'update') {
			const { record } = currentState;
			result = await runner.run(crud.update(record.id, data));
			ui.toast.success(m.Record_updated_successfully());
		} else {
			return;
		}
		await onSuccess?.(currentState.mode, result);
	}

	async changeMode(mode: State<C>) {
		this.form.superform?.reset();
		this.currentState = mode;

		let input: Partial<pb.BaseRecord<C>>;
		if (mode.mode === 'create') {
			input = mode.initialData;
		} else {
			input = mode.record;
		}

		const formData = recordToFormData(this.config.collection, input) as Partial<FormData>;
		await this.form.update(formData, { validate: false });
	}
}
