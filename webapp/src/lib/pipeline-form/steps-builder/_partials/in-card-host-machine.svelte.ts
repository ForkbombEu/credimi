// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { tick } from 'svelte';
import { FiniteStateMachine } from 'runed';

import { playInCardEnterLayout, playInCardExitLayout } from '../in-card-layout.js';
import { cancelMotion, type MotionHandle } from './in-card-motion.js';

export type InCardLayoutState = 'idle' | 'waiting' | 'entering' | 'settled' | 'exiting';

export type InCardLayoutEls = {
	card: HTMLElement;
	lock: HTMLElement;
	display: HTMLElement;
	form: HTMLElement;
};

type InCardLayoutEvent = 'hold' | 'ready' | 'settled' | 'release' | 'done' | 'drop' | 'abort';

type LayoutFacts = { showFormBody: boolean; expandReady: boolean; editing: boolean };

function completeEls(getEls: () => InCardLayoutEls | null): InCardLayoutEls | null {
	const els = getEls();
	if (!els?.card || !els.lock || !els.display || !els.form) return null;
	return els;
}

function noop(): void {}

export function createInCardLayoutMachine(options: {
	getEls: () => InCardLayoutEls | null;
	getCardFillMaxPx: () => number | undefined;
	onDone: () => void;
}): {
	get current(): InCardLayoutState;
	get bodyMaxPx(): number | undefined;
	sync(facts: LayoutFacts): void;
	destroy(): void;
} {
	const { getEls, getCardFillMaxPx, onDone } = options;

	let phase = $state<InCardLayoutState>('idle');
	let bodyMax = $state<number | undefined>(undefined);
	let handle: MotionHandle | undefined;

	function cancelLayerMotion(): void {
		const els = getEls();
		cancelMotion(els?.lock);
		cancelMotion(els?.display);
		cancelMotion(els?.form);
	}

	function cancelHandle(): void {
		handle?.cancel();
		handle = undefined;
	}

	function finishExitIfCurrent(): void {
		if (fsm.current !== 'exiting') return;
		fsm.send('done');
		if (fsm.current === 'idle') onDone();
	}

	function playExit(enterSettled: boolean, els: InCardLayoutEls | null): void {
		handle = playInCardExitLayout({
			...(els && enterSettled
				? { lock: els.lock, display: els.display, form: els.form }
				: {}),
			enterSettled,
			onComplete: finishExitIfCurrent
		});
	}

	const fsm = new FiniteStateMachine<InCardLayoutState, InCardLayoutEvent>('idle', {
		idle: {
			_enter(meta) {
				phase = 'idle';
				if (meta.from === null) return;
				bodyMax = undefined;
				cancelHandle();
				cancelLayerMotion();
			},
			hold: 'waiting',
			drop: 'idle',
			release: noop
		},
		waiting: {
			_enter() {
				phase = 'waiting';
			},
			hold: noop,
			ready: () => (completeEls(getEls) ? 'entering' : undefined),
			release: 'exiting',
			drop: 'idle'
		},
		entering: {
			_enter() {
				phase = 'entering';
				const els = completeEls(getEls);
				if (!els) return;
				handle = playInCardEnterLayout({
					...els,
					cardFillMaxPx: getCardFillMaxPx(),
					onSettled: ({ bodyMaxPx: settled }) => {
						if (fsm.current !== 'entering') return;
						bodyMax = settled;
						fsm.send('settled');
					}
				});
			},
			_exit() {
				cancelHandle();
				cancelLayerMotion();
			},
			hold: noop,
			ready: noop,
			settled: 'settled',
			release: 'exiting',
			drop: 'idle'
		},
		settled: {
			_enter() {
				phase = 'settled';
			},
			hold: noop,
			ready: noop,
			release: 'exiting',
			drop: 'idle'
		},
		exiting: {
			_enter(meta) {
				phase = 'exiting';
				const fromSettled = meta.from === 'settled';
				if (!fromSettled) {
					playExit(false, completeEls(getEls));
					return;
				}
				void tick().then(() => {
					if (fsm.current !== 'exiting') return;
					const els = completeEls(getEls);
					playExit(els != null, els);
				});
			},
			_exit() {
				cancelHandle();
				cancelLayerMotion();
			},
			hold: noop,
			ready: noop,
			release: noop,
			done: 'idle',
			drop: 'idle'
		},
		'*': {
			abort: 'idle'
		}
	});

	return {
		get current() {
			return phase;
		},
		get bodyMaxPx() {
			return bodyMax;
		},
		sync(facts: LayoutFacts) {
			if (!facts.showFormBody) {
				fsm.send('drop');
				return;
			}
			fsm.send('hold');
			if (facts.expandReady && facts.editing && completeEls(getEls)) {
				fsm.send('ready');
			} else if (!facts.editing) {
				fsm.send('release');
			}
		},
		destroy() {
			fsm.send('abort');
		}
	};
}
