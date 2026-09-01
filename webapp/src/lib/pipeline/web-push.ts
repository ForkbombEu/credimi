// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { ClientResponseError } from 'pocketbase';
import { err, ok, type Result } from 'true-myth/result';

import { m } from '@/i18n';
import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors';

//

export type NotificationState = 'unsupported' | 'denied' | 'off' | 'on';

type PushSubscriptionKeys = {
	p256dh: string;
	auth: string;
};

const VAPID_PUBLIC_KEY_URL = '/api/web-push/vapid-public-key';

export function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
	const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
	const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
	const rawData = atob(base64);
	const output = new Uint8Array(rawData.length);
	for (let i = 0; i < rawData.length; ++i) output[i] = rawData.charCodeAt(i);
	return output;
}

export async function getNotificationState(): Promise<NotificationState> {
	if (!browser) return 'unsupported';
	if (!('serviceWorker' in navigator && 'PushManager' in window)) return 'unsupported';
	if (Notification.permission === 'denied') return 'denied';

	// `getRegistration` resolves immediately (undefined when no worker exists),
	// unlike `ready` which waits forever, so the profile page can never hang.
	const registration = await navigator.serviceWorker.getRegistration();
	if (!registration) return 'off';

	try {
		return (await registration.pushManager.getSubscription()) ? 'on' : 'off';
	} catch {
		// The registration can exist before the worker is active,
		// in which case `getSubscription` throws.
		return 'off';
	}
}

export async function enablePipelineNotifications(): Promise<Result<null, string>> {
	if (!browser) return err(m.Push_notifications_unsupported());

	try {
		if (!('serviceWorker' in navigator && 'PushManager' in window))
			return err(m.Push_notifications_unsupported());

		const permission = await Notification.requestPermission();
		if (permission !== 'granted') return err(m.Failed_to_enable_pipeline_notifications());
		// SvelteKit registers the service worker automatically, but fall
		// back to an explicit registration so the toggle cannot hang on
		// `ready` if the automatic registration has not happened yet.
		const registration =
			(await navigator.serviceWorker.getRegistration()) ??
			(await navigator.serviceWorker.register('/service-worker.js'));
		let subscription = await registration.pushManager.getSubscription();
		if (!subscription) {
			const { public_key } = await pb.send<{ public_key: string }>(VAPID_PUBLIC_KEY_URL, {
				method: 'GET'
			});
			subscription = await registration.pushManager.subscribe({
				userVisibleOnly: true,
				applicationServerKey: urlBase64ToUint8Array(public_key)
			});
		}

		await saveSubscription(subscription);
		return ok(null);
	} catch (e) {
		return err(getExceptionMessage(e));
	}
}

export async function disablePipelineNotifications(): Promise<Result<null, string>> {
	if (!browser) return ok(null);

	try {
		const registration = await navigator.serviceWorker.getRegistration();
		const subscription = await registration?.pushManager.getSubscription();
		if (subscription) {
			await deleteSubscriptionRecords(subscription.endpoint);
			await subscription.unsubscribe();
		}
		return ok(null);
	} catch (e) {
		return err(getExceptionMessage(e));
	}
}

async function saveSubscription(subscription: PushSubscription): Promise<void> {
	const { endpoint, keys } = subscription.toJSON() as {
		endpoint?: string;
		keys?: PushSubscriptionKeys;
	};
	const user = pb.authStore.record?.id;
	if (!user || !endpoint || !keys?.p256dh || !keys.auth) {
		throw new Error('Missing push subscription data');
	}

	try {
		await pb.collection('push_subscriptions').create({ user, endpoint, keys });
	} catch (e) {
		// A unique-constraint conflict on `endpoint` means this browser is
		// already registered: refresh its keys instead of failing.
		if (!(e instanceof ClientResponseError)) throw e;
		const user = pb.authStore.record?.id;
		if (!user) throw e;
		const existing = await findSubscriptionRecord(user, endpoint);
		if (!existing) throw e;
		await pb.collection('push_subscriptions').update(existing.id, { user, keys });
	}
}

async function deleteSubscriptionRecords(endpoint: string): Promise<void> {
	const user = pb.authStore.record?.id;
	if (!user) return;

	const record = await findSubscriptionRecord(user, endpoint);

	if (record) {
		await pb.collection('push_subscriptions').delete(record.id);
		return;
	}

	// No record matches this endpoint (e.g. the browser lost the subscription):
	// clear all of the user's records so no stale subscription keeps pushing.
	const records = await pb.collection('push_subscriptions').getFullList({
		filter: pb.filter('user = {:user}', { user })
	});
	await Promise.all(records.map((r) => pb.collection('push_subscriptions').delete(r.id)));
}

async function findSubscriptionRecord(user: string, endpoint: string) {
	return pb
		.collection('push_subscriptions')
		.getFirstListItem(pb.filter('user = {:user} && endpoint = {:endpoint}', { user, endpoint }))
		.catch(() => null);
}
