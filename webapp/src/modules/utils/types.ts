// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Base types

export type GenericRecord = Record<string, unknown>;
export type StringRecord = Record<string, string>;

export type MaybePromise<T> = T | Promise<T>;

// Logic operations

export type If<Condition extends boolean, IfTrue, IfFalse> = Condition extends true
	? IfTrue
	: IfFalse;

export type Not<Condition extends boolean> = Condition extends true ? false : true;

//

export type IsArray<T> = T extends Array<unknown> ? true : false;

export type KeyOf<T> = Extract<keyof T, string>;

export type ValueOf<T> = T[keyof T];

export type InferArrayType<T> = T extends (infer U)[] ? U : T;

export type StringKey<R extends Record<string, unknown>> = {
	[K in keyof R]: R[K] extends string ? K : never;
}[keyof R];

// Reactivity

export type State<T> = {
	current: T;
};

export type Getter<T> = () => T;
