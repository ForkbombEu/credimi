// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/* Window */

type Content = object;
type BeforeClose = (preventDefault: () => void) => void;

export class Window<C extends Content = object> {
	constructor(private readonly init: { content?: C; beforeClose?: BeforeClose } = {}) {}

	isOpen = $state(false);

	open() {
		this.isOpen = true;
	}

	close() {
		let prevent = false;
		const preventDefault = () => {
			prevent = true;
		};
		try {
			this.init?.beforeClose?.(preventDefault);
		} catch (e) {
			console.warn(e);
		}
		if (!prevent) this.isOpen = false;
	}
}

/* Alert */

type AlertAction = () => void | Promise<void>;

export class Alert<C extends Content = object, W extends Window<C> = Window<C>> {
	constructor(
		private readonly init: { window: W; onConfirm?: AlertAction; onDismiss?: AlertAction }
	) {}

	get window() {
		return this.init.window;
	}

	async confirm() {
		await this.init.onConfirm?.();
		this.init.window.close();
	}

	async dismiss() {
		await this.init.onDismiss?.();
		this.init.window.close();
	}
}
