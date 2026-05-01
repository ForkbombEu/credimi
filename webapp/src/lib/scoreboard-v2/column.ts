// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Component } from 'svelte';

import { createColumnHelper, type DisplayColumnDef } from '@tanstack/table-core';

import {
	RenderComponentConfig,
	RenderSnippetConfig,
	renderComponent
} from '@/components/ui/data-table/render-helpers';

import type { ScoreboardRow } from './types';

/* Base types */

type Accessor = (row: ScoreboardRow) => unknown;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type HeaderConfig = RenderComponentConfig<Component<any>> | RenderSnippetConfig<Component<any>>;

/* Core cell config */

type Config<A extends Accessor> = {
	fn: A;
	id: string;
	header?: HeaderConfig | string;
};

export function define<A extends Accessor>(config: Config<A>): Config<A> {
	return config;
}

/* Cell component props */

type BaseProps<A extends Accessor> = { value: ReturnType<A> };

export type Props<C extends Config<Accessor>> = BaseProps<C['fn']>;

/* Module management */

type Module<A extends Accessor> = {
	column: Config<A>;
	default: Component<BaseProps<A>>;
};

const helper = createColumnHelper<ScoreboardRow>();

export function build<A extends Accessor>(mod: Module<A>) {
	const config: DisplayColumnDef<ScoreboardRow, unknown> = {
		id: mod.column.id,
		cell: (info: { getValue: () => unknown }) => {
			return renderComponent(mod.default, { value: info.getValue() as ReturnType<A> });
		}
	};
	if (mod.column.header) {
		if (typeof mod.column.header === 'string') {
			config.header = mod.column.header;
		} else {
			config.header = () => mod.column.header;
		}
	}
	return helper.accessor(mod.column.fn, config);
}
