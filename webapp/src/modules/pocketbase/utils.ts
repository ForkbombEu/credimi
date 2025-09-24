// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { getCollectionModel } from '@/pocketbase/collections-models';
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

interface UniqueConstraint {
	fields: string[];
	indexName: string;
}

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
 * Parse unique constraints from a collection's indexes
 * @param collectionName Collection to analyze
 * @returns UniqueConstraint[] Array of unique constraints
 */
function parseUniqueConstraints(collectionName: CloneableCollections): UniqueConstraint[] {
	try {
		const model = getCollectionModel(collectionName);
		const indexesString = typeof model.indexes === 'string' ? model.indexes : '[]';
		const indexes: string[] = JSON.parse(indexesString);

		const uniqueConstraints: UniqueConstraint[] = [];

		for (const indexDef of indexes) {
			if (indexDef.includes('UNIQUE INDEX')) {
				// Extract index name and fields from SQL
				// Example: "CREATE UNIQUE INDEX `idx_name` ON `table` (`field1`, `field2`)"
				const indexNameMatch = indexDef.match(/UNIQUE INDEX `([^`]+)`/);
				const fieldsMatch = indexDef.match(/\(([^)]+)\)/);

				if (indexNameMatch && fieldsMatch) {
					const indexName = indexNameMatch[1];
					const fieldsStr = fieldsMatch[1];

					// Parse field names, handling newlines and backticks
					const fields = fieldsStr
						.split(',')
						.map((field) => field.trim().replace(/[`\n]/g, ''))
						.filter((field) => field.length > 0);

					uniqueConstraints.push({
						indexName,
						fields
					});
				}
			}
		}

		return uniqueConstraints;
	} catch (error) {
		console.error(`Error parsing unique constraints for ${collectionName}:`, error);
		return [];
	}
}

/**
 * Check if a combination of field values is unique within constraints
 * @param collectionName Collection to check
 * @param constraint Unique constraint to validate
 * @param fieldValues Field values to check
 * @param excludeId Record ID to exclude from uniqueness check
 * @returns Promise<boolean> True if combination is unique
 */
async function isConstraintUnique(
	collectionName: CloneableCollections,
	constraint: UniqueConstraint,
	fieldValues: Record<string, unknown>,
	excludeId?: string
): Promise<boolean> {
	try {
		// Build filter for all fields in the constraint
		const filterParts = constraint.fields
			.map((field) => {
				const value = fieldValues[field];
				if (typeof value === 'string') {
					return `${field} = "${value}"`;
				} else if (value !== undefined && value !== null) {
					return `${field} = ${value}`;
				}
				return null;
			})
			.filter((part) => part !== null);

		if (filterParts.length === 0) return true;

		let filter = filterParts.join(' && ');
		if (excludeId) {
			filter += ` && id != "${excludeId}"`;
		}

		const records = await pb.collection(collectionName).getList(1, 1, { filter });
		return records.totalItems === 0;
	} catch (error) {
		console.error('Error checking constraint uniqueness:', error);
		return false;
	}
}

/**
 * Generate unique field values with display-field-first approach
 * Always modifies display field for UX, then handles constraint fields for database compliance
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

	// Step 2: Handle database constraint fields
	const uniqueConstraints = parseUniqueConstraints(collectionName);

	for (const constraint of uniqueConstraints) {
		// Check if this constraint is violated with our current data
		const isUnique = await isConstraintUnique(collectionName, constraint, updatedData);

		if (!isUnique) {
			// Find constraint fields that need modification (exclude already modified display field)
			const constraintFields = constraint.fields.filter(
				(field) =>
					field !== displayField && // Don't modify display field again
					(field === 'name' || field === 'uid' || field === 'key') && // Only modifiable fields
					updatedData[field] // Field has a value
			);

			if (constraintFields.length > 0) {
				// Modify the first suitable constraint field
				const fieldToModify = constraintFields[0];
				const originalValue = originalData[fieldToModify] as string;

				if (originalValue) {
					const constraintValue = await generateUniqueFieldValue(
						collectionName,
						fieldToModify,
						originalValue,
						baseSuffix
					);
					updatedData[fieldToModify] = constraintValue;
				}
			}
		}
	}

	// Step 3: For collections without constraints, fall back to legacy approach (but display field already handled)
	if (uniqueConstraints.length === 0) {
		// Apply legacy uniqueness to other important fields (uid, key) that weren't the display field
		const fieldsToCheck = ['uid', 'key'].filter((field) => field !== displayField);

		const parentFieldMap = {
			wallet_actions: 'wallet',
			credentials: 'credential_issuer',
			use_cases_verifications: 'verifier'
		} as const;

		const parentField = parentFieldMap[collectionName];
		const parentId = updatedData[parentField] as string;

		if (parentId) {
			for (const fieldName of fieldsToCheck) {
				if (updatedData[fieldName]) {
					const uniqueValue = await generateUniqueFieldValue(
						collectionName,
						fieldName,
						updatedData[fieldName] as string,
						baseSuffix
					);
					updatedData[fieldName] = uniqueValue;
				}
			}
		}
	}

	return updatedData;
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
		const uniqueCloneData = await generateUniqueFieldValues(
			collectionName,
			cloneData,
			originalRecord,
			'_clone'
		);

		// 5. Create the cloned record
		return await pb.collection(collectionName).create(uniqueCloneData);
	} catch (error) {
		console.error(`Error cloning ${collectionName} record:`, error);
		throw error;
	}
}
