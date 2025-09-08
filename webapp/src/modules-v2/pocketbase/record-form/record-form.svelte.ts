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

	// Fields options
	exclude?: Array<pb.Field<C> | { [K in Mode]?: pb.Field<C>[] }>;
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

export class Instance<C extends db.CollectionName> {
	runner = new task.Runner();
	crud: pocketbaseCrud.Instance<C>;

	private _form?: form.Instance<db.CollectionFormData[C]>;
	get form() {
		return this._form;
	}

	private currentState = $state<State<C>>();
	get currentMode() {
		return this.currentState?.mode;
	}

	constructor(private readonly config: Config<C>) {
		const { collection, crud, ...state } = config;
		this.currentState = state;
		this.crud = crud ?? new pocketbaseCrud.Instance(collection);
		this._form = this.buildForm(state);
	}

	async submit() {
		await this.form?.submit();
	}

	async setMode(mode: State<C>) {
		this.currentState = mode;
		this._form = this.buildForm(mode);
	}

	/* Utils */

	private buildForm(s: State<C>) {
		const { runner, crud } = this;
		const { onSuccess } = this.config;

		let input: Partial<pb.BaseRecord<C>>;
		if (s.mode === 'create') {
			input = s.initialData;
		} else {
			input = s.record;
		}

		return new form.Instance({
			adapter: this.buildAdapter(s.mode),
			initialData: recordToFormData(this.config.collection, input),
			onSubmit: async (data) => {
				let result: db.CollectionResponses[C];
				if (s.mode === 'create') {
					result = await runner.run(crud.create(data));
					ui.toast.success(m.Record_created_successfully());
				} else {
					result = await runner.run(crud.update(s.record.id, data));
					ui.toast.success(m.Record_updated_successfully());
				}
				await onSuccess?.(s.mode, result);
			},
			onError: (error) => {
				ui.toast.error(error.message);
			}
		});
	}

	private buildAdapter(mode: Mode): ValidationAdapter<db.CollectionFormData[C]> {
		const exclude = this.buildExcludedFields(mode);
		const schema = createCollectionZodSchema(this.config.collection).omit(
			// @ts-expect-error - TODO: fix this
			Object.fromEntries(exclude.map((key) => [key, true]))
		);
		return zod(schema) as unknown as ValidationAdapter<db.CollectionFormData[C]>;
	}

	private buildExcludedFields(mode: Mode): pb.Field<C>[] {
		const { exclude: excludeOption = [] } = this.config;
		const exclude: pb.Field<C>[] = [];
		for (const item of excludeOption) {
			if (typeof item === 'string') {
				exclude.push(item);
			} else if (mode in item && item[mode]) {
				exclude.push(...item[mode]);
			}
		}
		return exclude;
	}
}
