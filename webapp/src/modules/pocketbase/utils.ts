// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import {
	Collections,
	type CollectionResponses,
	type CredentialIssuersRecord,
	type CredentialsRecord,
	type UseCasesVerificationsRecord,
	type UsersResponse,
	type VerifiersRecord,
	type WalletActionsRecord,
	type WalletsRecord
} from '@/pocketbase/types';

export function getUserDisplayName(user: UsersResponse) {
	return user.name ? user.name : user.username ? user.username : user.email;
}

// Configuration mapping collections to their parent field names using type-safe field extraction
const COLLECTION_PARENT_FIELD_MAP = {
	[Collections.CredentialIssuers]: 'owner' as keyof CredentialIssuersRecord,
	[Collections.Credentials]: 'credential_issuer' as keyof CredentialsRecord,
	[Collections.Verifiers]: 'owner' as keyof VerifiersRecord,
	[Collections.UseCasesVerifications]: 'verifier' as keyof UseCasesVerificationsRecord,
	[Collections.Wallets]: 'owner' as keyof WalletsRecord,
	[Collections.WalletActions]: 'wallet' as keyof WalletActionsRecord
} as const;

/**
 * Check if a name is unique within a collection's parent scope
 *
 * This function validates name uniqueness for hierarchical collections where records
 * belong to a parent entity (e.g., credentials belong to credential_issuers).
 *
 * @param collectionName - Collection to check uniqueness within
 * @param name - Name value to check for uniqueness
 * @param parentId - ID of the parent record to scope the uniqueness check
 * @param excludeId - Optional record ID to exclude from the check (useful for updates)
 * @returns Promise<boolean> - True if name is unique within the parent scope, false otherwise
 *
 * @example
 * ```typescript
 * // Check if credential name is unique within a credential issuer
 * const isUnique = await checkNameUniqueness('credentials', 'My Credential', 'issuer123');
 *
 * // Check uniqueness while excluding current record (for updates)
 * const isUniqueForUpdate = await checkNameUniqueness('credentials', 'Updated Name', 'issuer123', 'record456');
 * ```
 *
 * @throws Will log errors but returns false on failure for safety
 */
export async function checkNameUniqueness(
	collectionName: keyof typeof COLLECTION_PARENT_FIELD_MAP,
	name: string,
	parentId: string,
	excludeId?: string
): Promise<boolean> {
	const parentField = COLLECTION_PARENT_FIELD_MAP[collectionName];

	try {
		const filter = excludeId
			? `${parentField} = "${parentId}" && name = "${name}" && id != "${excludeId}"`
			: `${parentField} = "${parentId}" && name = "${name}"`;

		const records = await pb.collection(collectionName).getList(1, 1, { filter });
		return records.totalItems === 0;
	} catch (error) {
		// Check if this is an auto-cancellation error
		if (error && typeof error === 'object' && 'isAbort' in error && error.isAbort) {
			console.log('Name uniqueness check was cancelled (auto-cancellation)');
			return true; // Assume unique when cancelled to avoid blocking the user
		}

		console.error('Error checking name uniqueness:', error);
		return false; // Assume not unique on other errors for safety
	}
}

/**
 * Get the field that's displayed to users in the UI
 * @param collectionName Collection to check
 * @param recordData Record data to analyze
 * @returns string The field name that users see
 */
function getDisplayField(
	collectionName: CloneableCollections,
	recordData: Record<string, unknown>
): string {
	const config = COLLECTION_DISPLAY_CONFIG[collectionName];

	// Try primary display field first
	if (
		recordData[config.primary] &&
		typeof recordData[config.primary] === 'string' &&
		(recordData[config.primary] as string).trim()
	) {
		return config.primary;
	}

	// Try fallback fields in order
	for (const fallback of config.fallbacks) {
		if (
			recordData[fallback] &&
			typeof recordData[fallback] === 'string' &&
			(recordData[fallback] as string).trim()
		) {
			return fallback;
		}
	}

	// Ultimate fallback to primary field even if empty
	return config.primary;
}

type SystemFields = 'id' | 'created' | 'updated';
type CloneableCollections = 'wallet_actions' | 'credentials' | 'use_cases_verifications';

// Configuration for display fields - easy to update when UI changes
const COLLECTION_DISPLAY_CONFIG = {
	wallet_actions: {
		primary: 'name',
		fallbacks: []
	},
	credentials: {
		primary: 'name',
		fallbacks: ['key']
	},
	use_cases_verifications: {
		primary: 'name',
		fallbacks: []
	}
} as const;

interface CloneOptions<T extends CloneableCollections> {
	/** Fields to exclude from cloning (in addition to system fields) */
	excludeFields?: Array<keyof CollectionResponses[T]>;
}

/**
 * Check if a field value is unique within a collection's parent scope
 * Simple wrapper around the existing parent-field uniqueness pattern
 * @param collectionName Collection to check
 * @param fieldName Field to check (name, key, uid, etc.)
 * @param fieldValue Value to check for uniqueness
 * @param parentId Parent record ID to scope the check
 * @returns Promise<boolean> True if field value is unique within parent scope
 */
async function checkFieldUniqueness(
	collectionName: CloneableCollections,
	fieldName: string,
	fieldValue: string,
	parentId: string
): Promise<boolean> {
	const parentFieldMap = {
		wallet_actions: 'wallet',
		credentials: 'credential_issuer',
		use_cases_verifications: 'verifier'
	} as const;

	const parentField = parentFieldMap[collectionName];

	try {
		const filter = `${parentField} = "${parentId}" && ${fieldName} = "${fieldValue}"`;
		const records = await pb.collection(collectionName).getList(1, 1, { filter });
		return records.totalItems === 0;
	} catch (error) {
		console.error('Error checking field uniqueness:', error);
		return false; // Assume not unique on error for safety
	}
}

/**
 * Generate unique field values with display-field-first approach
 * Always modifies display field for UX, then handles key constraint fields for database compliance
 * @param collectionName Collection to check against
 * @param cloneData Current clone data object
 * @param originalData Original record data
 * @param baseSuffix Base suffix to use (e.g., '_copy')
 * @returns Promise<Record<string, unknown>> Updated clone data with unique values
 */
async function generateUniqueFieldValues(
	collectionName: CloneableCollections,
	cloneData: Record<string, unknown>,
	originalData: Record<string, unknown>,
	baseSuffix: string = '_copy'
): Promise<Record<string, unknown>> {
	const updatedData = { ...cloneData };

	// Step 1: Always modify the display field first (for UX clarity)
	const displayField = getDisplayField(collectionName, originalData);
	if (updatedData[displayField]) {
		const displayValue = await generateUniqueFieldValue(
			collectionName,
			displayField,
			updatedData[displayField] as string,
			baseSuffix
		);
		updatedData[displayField] = displayValue;
	}

	// Step 2: Handle key constraint fields using existing checkNameUniqueness logic
	const parentFieldMap = {
		wallet_actions: 'wallet',
		credentials: 'credential_issuer',
		use_cases_verifications: 'verifier'
	} as const;

	const parentField = parentFieldMap[collectionName];
	const parentId = updatedData[parentField] as string;

	if (parentId) {
		// Check and fix uniqueness for important fields (excluding already modified display field)
		const fieldsToCheck = ['name', 'key', 'uid'].filter(
			(field) => field !== displayField && updatedData[field]
		);

		for (const fieldName of fieldsToCheck) {
			const fieldValue = updatedData[fieldName] as string;

			// Use existing checkNameUniqueness approach but for any field
			const isUnique = await checkFieldUniqueness(
				collectionName,
				fieldName,
				fieldValue,
				parentId
			);

			if (!isUnique) {
				const uniqueValue = await generateUniqueFieldValue(
					collectionName,
					fieldName,
					fieldValue,
					baseSuffix
				);
				updatedData[fieldName] = uniqueValue;
			}
		}
	}

	return updatedData;
}

/**
 * Check if error is a PocketBase unique constraint validation error
 * @param error Error object from PocketBase
 * @returns boolean True if it's a uniqueness validation error
 */
function isUniqueConstraintError(error: unknown): boolean {
	if (!error || typeof error !== 'object') return false;

	const errorObj = error as Record<string, unknown>;
	const response = errorObj?.response as Record<string, unknown> | undefined;
	const status = errorObj?.status;
	const errorData = response?.data;

	// Check if it's a 400 error with field-level validation data
	if (status !== 400 || !errorData || typeof errorData !== 'object') return false;

	// Check if any field has a unique constraint violation
	const fieldErrors = errorData as Record<string, unknown>;
	return Object.values(fieldErrors).some((fieldError) => {
		if (!fieldError || typeof fieldError !== 'object') return false;
		const error = fieldError as Record<string, unknown>;
		return error.code === 'validation_not_unique';
	});
}

/**
 * Extract field names that failed uniqueness validation
 * @param error Error object from PocketBase
 * @returns string[] Array of field names that failed uniqueness check
 */
function getFailedUniqueFields(error: unknown): string[] {
	if (!error || typeof error !== 'object') return [];

	const errorObj = error as Record<string, unknown>;
	const response = errorObj?.response as Record<string, unknown> | undefined;
	const errorData = response?.data;

	if (!errorData || typeof errorData !== 'object') return [];

	const failedFields: string[] = [];
	const fieldErrors = errorData as Record<string, unknown>;

	for (const [fieldName, fieldError] of Object.entries(fieldErrors)) {
		if (!fieldError || typeof fieldError !== 'object') continue;
		const error = fieldError as Record<string, unknown>;
		if (error.code === 'validation_not_unique') {
			failedFields.push(fieldName);
		}
	}

	return failedFields;
}

/**
 * Generate a unique field value by appending incremental suffixes
 * @param collectionName Collection to check against
 * @param fieldName Field to make unique
 * @param originalValue Original field value
 * @param baseSuffix Base suffix to use (e.g., '_copy')
 * @returns Promise<string> Unique field value
 */
async function generateUniqueFieldValue(
	collectionName: CloneableCollections,
	fieldName: string,
	originalValue: string,
	baseSuffix: string = '_copy'
): Promise<string> {
	let counter = 1;
	let candidateValue = `${originalValue}${baseSuffix}_${counter}`;

	try {
		// Keep incrementing until we find a unique value
		while (true) {
			const filter = `${fieldName} = "${candidateValue}"`;
			const records = await pb.collection(collectionName).getList(1, 1, { filter });

			if (records.totalItems === 0) {
				return candidateValue;
			}

			counter++;
			candidateValue = `${originalValue}${baseSuffix}_${counter}`;
		}
	} catch (error) {
		console.error('Error generating unique field value:', error);
		// Fallback: append timestamp
		return `${originalValue}${baseSuffix}_${Date.now()}`;
	}
}

/**
 * Clone a record with automatic schema-aware uniqueness handling
 * @param collectionName Collection to clone from
 * @param recordId ID of record to clone
 * @param options Clone options
 * @returns Promise<CollectionResponses[T]> Cloned record
 */
export async function cloneRecord<T extends CloneableCollections>(
	collectionName: T,
	recordId: string,
	options: CloneOptions<T> = {}
): Promise<CollectionResponses[T]> {
	const { excludeFields = [] } = options;

	// Prepare clone data once to be used in try/catch
	let uniqueCloneData: Record<string, unknown> | undefined;

	try {
		// 1. Fetch original record
		const originalRecord = await pb.collection(collectionName).getOne(recordId);

		// 2. Create clone data by removing system fields
		const systemFields: SystemFields[] = ['id', 'created', 'updated'];
		const fieldsToExclude = [...systemFields, ...excludeFields];

		const cloneData: Record<string, unknown> = {};

		for (const [key, value] of Object.entries(originalRecord)) {
			if (fieldsToExclude.includes(key as SystemFields)) continue;
			cloneData[key] = value;
		}

		// 3. Handle common transformations
		// Reset published status
		if ('published' in cloneData) {
			cloneData.published = false;
		}

		// 4. Apply schema-aware uniqueness handling
		uniqueCloneData = await generateUniqueFieldValues(
			collectionName,
			cloneData,
			originalRecord,
			'_clone'
		);

		// 5. Create the cloned record
		return await pb.collection(collectionName).create(uniqueCloneData);
	} catch (error: unknown) {
		// Check if it's a uniqueness constraint error that we can retry
		if (isUniqueConstraintError(error) && uniqueCloneData) {
			console.log('Uniqueness constraint error detected, attempting retry...');

			const failedFields = getFailedUniqueFields(error);
			console.log('Failed fields:', failedFields);

			// Filter to only fields we can modify (exclude parent references like credential_issuer)
			const parentFieldMap = {
				wallet_actions: 'wallet',
				credentials: 'credential_issuer',
				use_cases_verifications: 'verifier'
			} as const;

			const parentField = parentFieldMap[collectionName];
			const retryableFields = failedFields.filter(
				(field) =>
					field !== parentField && // Don't modify parent references
					['name', 'key', 'uid'].includes(field) && // Only modifiable fields
					uniqueCloneData &&
					field in uniqueCloneData // Field exists in our data
			);

			console.log('Retryable fields:', retryableFields);

			if (retryableFields.length > 0) {
				// Create updated data with unique values for failed fields
				const retryCloneData = { ...uniqueCloneData };

				for (const fieldName of retryableFields) {
					const originalValue = retryCloneData[fieldName] as string;
					const uniqueValue = await generateUniqueFieldValue(
						collectionName,
						fieldName,
						originalValue || '', // Handle empty strings
						'_clone'
					);
					retryCloneData[fieldName] = uniqueValue;
					console.log(`Updated ${fieldName}: "${originalValue}" → "${uniqueValue}"`);
				}

				// Retry the creation with updated data
				console.log('Retrying clone creation...');
				try {
					return await pb.collection(collectionName).create(retryCloneData);
				} catch (retryError) {
					console.error('Retry failed:', retryError);
					// Fall through to original error handling
				}
			} else {
				console.log('No retryable fields found, cannot fix constraint violations');
			}
		}

		// Check if it's a validation error with field details
		const errorObj = error as Record<string, unknown>;
		const response = errorObj?.response as Record<string, unknown> | undefined;
		const errorData = response?.data;
		if (errorData && typeof errorData === 'object') {
			console.log(
				'Field-level errors found:',
				Object.keys(errorData as Record<string, unknown>)
			);
			for (const [field, fieldError] of Object.entries(
				errorData as Record<string, unknown>
			)) {
				console.log(`  - Field "${field}":`, fieldError);
			}
		}

		console.error(`Error cloning ${collectionName} record:`, error);
		throw error;
	}
}
