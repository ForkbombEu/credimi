// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Component, ComponentProps } from 'svelte';

/**
 * Deferred component render payload: a component constructor plus its props.
 *
 * Render with:
 * ```svelte
 * {@const C = value.component}
 * <C {...value.props} />
 * ```
 *
 * Or pass into hosts that accept `Comp` / `Snippet` (e.g. StepCardDisplay footer).
 *
 * ## Bindings
 *
 * `bind:` / `$bindable` do **not** work through this bag. Spreading `props` is
 * one-way only. Prefer callback props (`onChange`, `onCheckedChange`, …).
 *
 * To preserve bindings, use real markup or a parent-scoped snippet instead:
 * ```svelte
 * {#snippet footer()}
 *   <Child bind:value={x} />
 * {/snippet}
 * ```
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export class Comp<TComponent extends Component<any> = Component<any>> {
	component: TComponent;
	props: ComponentProps<TComponent>;

	constructor(component: TComponent, props: ComponentProps<TComponent>) {
		this.component = component;
		this.props = props;
	}
}

/** Build a {@link Comp} for deferred `<C {...props} />` rendering. */
export function comp<
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	TComponent extends Component<any>
>(component: TComponent, props: ComponentProps<TComponent> = {} as ComponentProps<TComponent>) {
	return new Comp(component, props);
}
