// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { IsMobile } from "@/components/ui/hooks/is-mobile.svelte.js";

export class ResponsiveNav {
	#isMobile: IsMobile;
	#isOpen = $state(false);

	constructor() {
		this.#isMobile = new IsMobile();
	}

	get isMobile() {
		return this.#isMobile.current;
	}

	get isOpen() {
		return this.#isOpen;
	}

	setOpen = (open: boolean) => {
		this.#isOpen = open;
	};

	toggle = () => {
		this.#isOpen = !this.#isOpen;
	};

	close = () => {
		this.#isOpen = false;
	};
}
