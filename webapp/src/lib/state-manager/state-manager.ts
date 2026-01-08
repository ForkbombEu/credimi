// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { create } from 'mutative';

//

export class StateManager<State> {
	constructor(
		readonly getter: () => State,
		readonly setter: (state: State) => void
	) {}

	private history: History<State> = {
		past: [],
		future: []
	};

	run(action: (state: State) => void) {
		this.history.past.push(this.getter());
		const nextState = create(this.getter(), action);
		this.setter(nextState);
		this.history.future = [];
	}

	undo() {
		const previousData = this.history.past.pop();
		if (!previousData) return;
		this.history.future.push(this.getter());
		this.setter(previousData);
	}

	redo() {
		const nextState = this.history.future.pop();
		if (!nextState) return;
		this.history.past.push(this.getter());
		this.setter(nextState);
	}
}

type History<State> = {
	past: State[];
	future: State[];
};
