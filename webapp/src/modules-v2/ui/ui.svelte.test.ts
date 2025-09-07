// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, test, vi } from 'vitest';

import { Alert, Window } from './ui.svelte';

//

describe('Window', () => {
	let window: Window;

	beforeEach(() => {
		window = new Window();
	});

	test('starts with isOpen as false', () => {
		expect(window.isOpen).toBe(false);
	});

	test('open sets isOpen to true', () => {
		window.open();
		expect(window.isOpen).toBe(true);
	});

	test('close sets isOpen to false', () => {
		window.open();
		expect(window.isOpen).toBe(true);

		window.close();
		expect(window.isOpen).toBe(false);
	});

	test('close with no beforeClose handler works normally', () => {
		window.open();
		window.close();
		expect(window.isOpen).toBe(false);
	});

	test('beforeClose can prevent closing', () => {
		const beforeClose = vi.fn((preventDefault: () => void) => {
			preventDefault();
		});

		window = new Window({ beforeClose });
		window.open();
		expect(window.isOpen).toBe(true);

		window.close();
		expect(beforeClose).toHaveBeenCalled();
		expect(window.isOpen).toBe(true);
	});

	test('beforeClose can allow closing', () => {
		const beforeClose = vi.fn();

		window = new Window({ beforeClose });
		window.open();
		expect(window.isOpen).toBe(true);

		window.close();
		expect(beforeClose).toHaveBeenCalled();
		expect(window.isOpen).toBe(false);
	});

	test('beforeClose error is caught and logged', () => {
		const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
		const error = new Error('Test error');
		const beforeClose = vi.fn(() => {
			throw error;
		});

		window = new Window({ beforeClose });
		window.open();

		window.close();
		expect(beforeClose).toHaveBeenCalled();
		expect(consoleSpy).toHaveBeenCalledWith(error);
		expect(window.isOpen).toBe(false); // Should still close despite error

		consoleSpy.mockRestore();
	});

	test('window can be created with content', () => {
		const content = { title: 'Test Window', data: 123 };
		const windowWithContent = new Window({ content });

		expect(windowWithContent.isOpen).toBe(false);
		windowWithContent.open();
		expect(windowWithContent.isOpen).toBe(true);
	});

	test('setting isOpen to false triggers beforeClose', () => {
		const beforeClose = vi.fn();
		window = new Window({ beforeClose });
		window.open();
		expect(window.isOpen).toBe(true);
		window.isOpen = false;
		expect(beforeClose).toHaveBeenCalled();
	});

	test('setting isOpen to false triggers beforeClose with preventDefault', () => {
		const beforeClose = vi.fn((preventDefault: () => void) => {
			preventDefault();
		});
		window = new Window({ beforeClose });
		window.open();
		expect(window.isOpen).toBe(true);
		window.isOpen = false;
		expect(beforeClose).toHaveBeenCalled();
		expect(window.isOpen).toBe(true);
	});
});

describe('Alert', () => {
	let window: Window;
	let alert: Alert;
	let onConfirm: ReturnType<typeof vi.fn>;
	let onDismiss: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		window = new Window();
		onConfirm = vi.fn();
		onDismiss = vi.fn();
		alert = new Alert({ window, onConfirm, onDismiss });
	});

	test('exposes window instance', () => {
		expect(alert.window).toBe(window);
	});

	test('confirm calls onConfirm and closes window', async () => {
		window.open();
		expect(window.isOpen).toBe(true);

		await alert.confirm();

		expect(onConfirm).toHaveBeenCalled();
		expect(window.isOpen).toBe(false);
	});

	test('dismiss calls onDismiss and closes window', async () => {
		window.open();
		expect(window.isOpen).toBe(true);

		await alert.dismiss();

		expect(onDismiss).toHaveBeenCalled();
		expect(window.isOpen).toBe(false);
	});

	test('confirm works without onConfirm handler', async () => {
		alert = new Alert({ window });
		window.open();
		expect(window.isOpen).toBe(true);

		await alert.confirm();
		expect(window.isOpen).toBe(false);
	});

	test('dismiss works without onDismiss handler', async () => {
		alert = new Alert({ window });
		window.open();
		expect(window.isOpen).toBe(true);

		await alert.dismiss();
		expect(window.isOpen).toBe(false);
	});

	test('confirm with async onConfirm handler', async () => {
		const asyncOnConfirm = vi.fn().mockResolvedValue(undefined);
		alert = new Alert({ window, onConfirm: asyncOnConfirm });

		window.open();
		const confirmPromise = alert.confirm();

		expect(asyncOnConfirm).toHaveBeenCalled();
		await confirmPromise;
		expect(window.isOpen).toBe(false);
	});

	test('dismiss with async onDismiss handler', async () => {
		const asyncOnDismiss = vi.fn().mockResolvedValue(undefined);
		alert = new Alert({ window, onDismiss: asyncOnDismiss });

		window.open();
		const dismissPromise = alert.dismiss();

		expect(asyncOnDismiss).toHaveBeenCalled();
		await dismissPromise;
		expect(window.isOpen).toBe(false);
	});

	test('confirm handles async onConfirm errors gracefully', async () => {
		const error = new Error('Async error');
		const asyncOnConfirm = vi.fn().mockRejectedValue(error);
		alert = new Alert({ window, onConfirm: asyncOnConfirm });

		window.open();

		// Should not throw
		await expect(alert.confirm()).rejects.toThrow('Async error');
		expect(asyncOnConfirm).toHaveBeenCalled();
		// Window should still be open since confirm failed
		expect(window.isOpen).toBe(true);
	});

	test('dismiss handles async onDismiss errors gracefully', async () => {
		const error = new Error('Async error');
		const asyncOnDismiss = vi.fn().mockRejectedValue(error);
		alert = new Alert({ window, onDismiss: asyncOnDismiss });

		window.open();

		// Should not throw
		await expect(alert.dismiss()).rejects.toThrow('Async error');
		expect(asyncOnDismiss).toHaveBeenCalled();
		// Window should still be open since dismiss failed
		expect(window.isOpen).toBe(true);
	});
});
