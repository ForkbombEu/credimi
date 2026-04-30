// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Component, ComponentProps } from 'svelte';

import { createColumnHelper } from '@tanstack/table-core';

import {
	RenderComponentConfig,
	renderComponent
} from '@/components/ui/data-table/render-helpers';

import type { ScoreboardRow } from './types';

//

type Accessor = (row: ScoreboardRow) => unknown;

// Config

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type HeaderConfig = RenderComponentConfig<Component<any>>;

type Config<A extends Accessor> = {
	fn: A;
	id: string;
	header: HeaderConfig;
};

export function define<A extends Accessor>(config: Config<A>): Config<A> {
	return config;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function header<TComponent extends Component<any>>(
	component: TComponent,
	props: ComponentProps<TComponent>
): RenderComponentConfig<TComponent> {
	return renderComponent(component, props);
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
		header: () => mod.column.header,
		cell: (info: { getValue: () => unknown }) => {
			return renderComponent(mod.default, { value: info.getValue() as ReturnType<A> });
		}
	});
}
