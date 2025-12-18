// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { create } from 'mutative';

//

export class StateManager<State> {
	constructor(public state: State) {}

	private history: History<State> = {
		past: [],
		future: []
	};

	run(action: (state: State) => void) {
		this.history.past.push(this.state);
		const nextState = create(this.state, action);
		this.state = nextState;
		this.history.future = [];
	}

	undo() {
		const previousData = this.history.past.pop();
		if (!previousData) return;
		this.history.future.push(this.state);
		this.state = previousData;
	}

	redo() {
		const nextState = this.history.future.pop();
		if (!nextState) return;
		this.history.past.push(this.state);
		this.state = nextState;
	}
}

type History<State> = {
	past: State[];
	future: State[];
};
