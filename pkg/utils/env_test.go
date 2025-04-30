// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"os"
	"testing"
)

func Test_GetEnvironmentVariable(t *testing.T) {
	t.Run("test with an existing variable", func(t *testing.T) {
		envVar := GetEnvironmentVariable("PWD")
		if envVar == "" {
			t.Errorf("Expected a value for the environment variable PWD, got an empty string")
		}
	})

	t.Run("test with a non-existing variable", func(t *testing.T) {
		envVar := GetEnvironmentVariable("NON_EXISTING_ENV_VAR")
		if envVar != "" {
			t.Errorf(
				"Expected an empty string for the environment variable NON_EXISTING_ENV_VAR, got a value",
			)
		}
	})

	t.Run("pass default value to a non existing variable", func(t *testing.T) {
		got := GetEnvironmentVariable("NON_EXISTING_ENV_VAR", "default")
		want := "default"
		if got != want {
			t.Errorf(
				"Expected 'default' for the environment variable NON_EXISTING_ENV_VAR, got a value",
			)
		}
	})
	t.Run("test a required variable", func(t *testing.T) {
		envVar := GetEnvironmentVariable("PWD", "", true)
		if envVar == "" {
			t.Errorf("Expected a value for the environment variable PWD, got an empty string")
		}
	})

	t.Run("test a required variable non existing should return a error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()
		GetEnvironmentVariable("NON_EXISTING_ENV_VAR", "", true)
	})

	t.Run(
		"test a required  non existing variable with default value should not return a error",
		func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("The code panicked")
				}
			}()
			GetEnvironmentVariable("NON_EXISTING_ENV_VAR", "default", true)
		},
	)
}

func Test_GetEnvironmentVariableAsInteger(t *testing.T) {
	t.Run("test with an existing int variable should return an integer", func(t *testing.T) {
		err := os.Setenv("MY_ENV_VAR", "9")
		if err != nil {
			t.Errorf("Failed to set the environment variable")
		}
		var got any
		envVar, err := GetEnvironmentVariableAsInteger("MY_ENV_VAR", 0)
		got = envVar
		if err != nil {
			t.Errorf(
				"Expected a value for the environment variable MY_ENV_VAR, got an empty string",
			)
		}
		_, ok := got.(int)
		if !ok {
			t.Errorf(
				"Expected an integer for the environment variable MY_ENV_VAR, got a different type",
			)
		}
		if got != 9 {
			t.Errorf("Expected '9' for the environment variable MY_ENV_VAR, got a different value")
		}
	})

	t.Run("test with a non-existing variable", func(t *testing.T) {
		envVar, err := GetEnvironmentVariableAsInteger("NON_EXISTING_ENV_VAR", 0)
		if err != nil {
			t.Errorf(
				"Expected a value for the environment variable NON_EXISTING_ENV_VAR, got an empty string",
			)
		}
		if envVar != 0 {
			t.Errorf(
				"Expected an empty string for the environment variable NON_EXISTING_ENV_VAR, got a value",
			)
		}
	})

	t.Run("test a string variable should return a error", func(t *testing.T) {
		err := os.Setenv("MY_ENV_VAR", "hello")
		if err != nil {
			t.Errorf("Failed to set the environment variable")
		}
		envVar, err := GetEnvironmentVariableAsInteger("MY_ENV_VAR", 0)
		if err == nil {
			t.Errorf("Expected an error for the environment variable MY_ENV_VAR, got nil")
		}
		if envVar != 0 {
			t.Errorf(
				"Expected an empty string for the environment variable MY_ENV_VAR, got a value",
			)
		}
	})
}
