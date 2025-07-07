// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// import type { FieldConfig } from './logic';

// export const sharedFields: Array<FieldConfig & { for: string[] }> = [
// 	{
// 		field_name: 'alias',
// 		credimi_id: 'id_1',
// 		i18_label: 'alias',
// 		i18n_description: 'alias',
// 		field_type: 'string',
// 		for: ['test1', 'test2', 'test3']
// 	},
// 	{
// 		field_name: 'description',
// 		credimi_id: 'id_2',
// 		i18_label: 'description',
// 		i18n_description: 'description',
// 		field_type: 'object',
// 		for: ['test1', 'test3']
// 	},
// 	{
// 		field_name: 'client_id',
// 		credimi_id: 'id_3',
// 		i18_label: 'client_id',
// 		i18n_description: 'client_id',
// 		field_type: 'string',
// 		for: ['test2', 'test3']
// 	}
// ];

// export const testsFields: Record<string, Array<FieldConfig>> = {
// 	test1: [
// 		{
// 			field_name: 'alias',
// 			credimi_id: 'id_1',
// 			i18_label: 'alias',
// 			i18n_description: 'alias',
// 			field_type: 'string'
// 		},
// 		{
// 			field_name: 'description',
// 			credimi_id: 'id_2',
// 			i18_label: 'description',
// 			i18n_description: 'description',
// 			field_type: 'object'
// 		},
// 		{
// 			field_name: 'presentation_definition',
// 			credimi_id: 'id_4',
// 			i18_label: 'presentation_definition',
// 			i18n_description: 'presentation_definition',
// 			field_type: 'object'
// 		},
// 		{
// 			field_name: 'test1_unique_field',
// 			credimi_id: 'test1_unique_1',
// 			i18_label: 'test1_unique_field',
// 			i18n_description: 'A field unique to test1',
// 			field_type: 'string'
// 		}
// 	],
// 	test2: [
// 		{
// 			field_name: 'alias',
// 			credimi_id: 'id_1',
// 			i18_label: 'alias',
// 			i18n_description: 'alias',
// 			field_type: 'string'
// 		},
// 		{
// 			field_name: 'client_id',
// 			credimi_id: 'id_3',
// 			i18_label: 'client_id',
// 			i18n_description: 'client_id',
// 			field_type: 'string'
// 		},
// 		{
// 			field_name: 'jwks',
// 			credimi_id: 'id_5',
// 			i18_label: 'jwks',
// 			i18n_description: 'jwks',
// 			field_type: 'object'
// 		},
// 		{
// 			field_name: 'test2_config',
// 			credimi_id: 'test2_unique_1',
// 			i18_label: 'test2_config',
// 			i18n_description: 'Configuration specific to test2',
// 			field_type: 'object'
// 		}
// 	],
// 	test3: [
// 		{
// 			field_name: 'alias',
// 			credimi_id: 'id_1',
// 			i18_label: 'alias',
// 			i18n_description: 'alias',
// 			field_type: 'string'
// 		},
// 		{
// 			field_name: 'description',
// 			credimi_id: 'id_2',
// 			i18_label: 'description',
// 			i18n_description: 'description',
// 			field_type: 'object'
// 		},
// 		{
// 			field_name: 'client_id',
// 			credimi_id: 'id_3',
// 			i18_label: 'client_id',
// 			i18n_description: 'client_id',
// 			field_type: 'string'
// 		},
// 		{
// 			field_name: 'test3_unique_field',
// 			credimi_id: 'test3_unique_1',
// 			i18_label: 'test3 Specific Field',
// 			i18n_description: 'A field only used in test3',
// 			field_type: 'string'
// 		}
// 	]
// };

// export const testsConfigs: Record<string, Record<string, unknown>> = {
// 	test1: {
// 		variant: {
// 			credential_format: 'sd_jwt_vc',
// 			client_id_scheme: 'did',
// 			request_method: 'request_uri_signed',
// 			response_mode: 'direct_post'
// 		},
// 		form: {
// 			alias: '{{ id_1 }}',
// 			description: '{{ id_2 }}',
// 			server: {
// 				authorization_endpoint: 'openid-vc://'
// 			},
// 			client: {
// 				presentation_definition: '{{ id_4 }}',
// 				test1_unique_field: '{{ test1_unique_1 }}'
// 			}
// 		}
// 	},
// 	test2: {
// 		variant: {
// 			credential_format: 'sd_jwt_vc',
// 			client_id_scheme: 'did',
// 			request_method: 'request_uri_signed',
// 			response_mode: 'direct_post'
// 		},
// 		form: {
// 			alias: '{{ id_1 }}',
// 			server: {
// 				authorization_endpoint: 'openid-vc://'
// 			},
// 			client: {
// 				client_id: '{{ id_3 }}',
// 				jwks: '{{ id_5 }}',
// 				test2_config: '{{ test2_unique_1 }}'
// 			}
// 		}
// 	},
// 	test3: {
// 		variant: {
// 			credential_format: 'sd_jwt_vc',
// 			client_id_scheme: 'did',
// 			request_method: 'request_uri_signed',
// 			response_mode: 'direct_post'
// 		},
// 		form: {
// 			alias: '{{ id_1 }}',
// 			description: '{{ id_2 }}',
// 			server: {
// 				authorization_endpoint: 'openid-vc://'
// 			},
// 			client: {
// 				client_id: '{{ id_3 }}',
// 				test3_unique_field: '{{ test3_unique_1 }}'
// 			}
// 		}
// 	}
// };
