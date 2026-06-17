// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}

	interface TurnstileRenderOptions {
		sitekey: string;
		callback?: (token: string) => void;
		theme?: 'auto' | 'dark' | 'light';
	}

	interface Turnstile {
		render(container: HTMLElement, options: TurnstileRenderOptions): string;
		reset(widgetId: string): void;
	}

	interface Window {
		turnstile?: Turnstile;
	}
}

export {};
