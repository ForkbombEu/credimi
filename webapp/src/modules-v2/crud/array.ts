// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { crud, types as t, task } from '#';

//

interface DefaultConfig {
	type: object;
}

interface ResolvedConfig<C extends DefaultConfig = DefaultConfig> extends crud.DefaultConfig {
	type: C['type'];
	input: C['type'];
	key: number;
	keyName: 'index';
}

export class Instance<
	Config extends DefaultConfig,
	RConfig extends ResolvedConfig<Config> = ResolvedConfig<Config>,
	Type extends Config['type'] = Config['type']
> implements crud.Crud<RConfig>
{
	constructor(private readonly items: Type[] = []) {}

	read(index: number): crud.Task<RConfig> {
		const item = this.items.at(index);
		if (!item) return task.reject(new t.NotFoundError(index));
		return task.resolve({ index, ...item });
	}

	readAll(): task.Task<crud.Record<RConfig>[], t.BaseError> {
		return task.resolve(this.items.map((val, index) => ({ index, ...val })));
	}

	create(input: Type): crud.Task<RConfig> {
		const index = this.items.length;
		this.items.push(input);
		return task.resolve({ ...input, index });
	}

	update(index: number, input: Partial<Type>): crud.Task<RConfig> {
		const item = this.items.at(index);
		if (!item) return task.reject(new t.NotFoundError(index));
		return task.resolve({ index, ...item, ...input });
	}

	delete(key: number): task.Task<boolean, t.BaseError> {
		if (key < 0 || key >= this.items.length) return task.reject(new t.NotFoundError(key));
		// @ts-expect-error - we want to remove the item
		this.items[key] = undefined;
		return task.resolve(true);
	}
}
