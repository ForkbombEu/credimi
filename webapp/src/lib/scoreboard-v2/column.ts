// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Component } from 'svelte';

import { createColumnHelper } from '@tanstack/table-core';

import { renderComponent } from '@/components/ui/data-table';

import type { ScoreboardRow } from './types';

//

type Accessor = (row: ScoreboardRow) => unknown;

// Config

type Config<A extends Accessor> = {
	fn: A;
	id: string;
	header: string;
};

export function define<A extends Accessor>(config: Config<A>): Config<A> {
	return config;
}

// Component props

type BaseProps<A extends Accessor> = { value: ReturnType<A> };

export type Props<C extends Config<Accessor>> = BaseProps<C['fn']>;

// Module management

type Module<A extends Accessor> = {
	column: Config<A>;
	default: Component<BaseProps<A>>;
};

const helper = createColumnHelper<ScoreboardRow>();

export function build<A extends Accessor>(mod: Module<A>) {
	return helper.accessor(mod.column.fn, {
		id: mod.column.id,
		header: mod.column.header,
		cell: (info) => renderComponent(mod.default, { value: info.getValue() as ReturnType<A> })
	});
}
