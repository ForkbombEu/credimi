// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

// GetEnvironmentVariable retrieves the value of an environment variable.
//
// Parameters:
//   - name: The name of the environment variable to retrieve.
//   - defaultValue (if provided): The value to return if the environment variable is not set.
//   - required (if provided): A boolean indicating whether the environment variable is required.
//
// Returns:
//   - string: The value of the environment variable, or the default value if not set.
//     If the variable is required and not set, the function panics.
func GetEnvironmentVariable(name string, others ...any) string {
	var defaultValue = ""
	var required bool

	if len(others) > 0 {
		if val, ok := others[0].(string); ok {
			defaultValue = val
		}
	}
	if len(others) > 1 {
		if val, ok := others[1].(bool); ok {
			required = val
		}
	}

	output := os.Getenv(name)
	if output == "" {
		output = defaultValue
	}
	if output == "" && required {
		panic("The environment variable " + name + " is required")
	}
	return output
}

// GetEnvironmentVariableAsInteger retrieves the value of an environment variable and converts it to an integer.
//
// Parameters:
//   - name: The name of the environment variable to retrieve.
//   - others: Optional variadic parameters:
//   - First parameter (if provided): The default integer value to return if the environment variable is not set or empty.
//   - Second parameter (if provided): A boolean indicating whether the environment variable is required.
//
// Returns:
//   - int: The integer value of the environment variable, or the default value if not set or empty.
//   - error: An error if the environment variable cannot be parsed as an integer or if the value is out of range for int.
//     Returns nil if no error occurred.
func GetEnvironmentVariableAsInteger(name string, others ...any) (int, error) {
	var defaultValue = 0
	var required bool

	if len(others) > 0 {
		if val, ok := others[0].(int); ok {
			defaultValue = val
		}
	}
	if len(others) > 1 {
		if val, ok := others[1].(bool); ok {
			required = val
		}
	}

	output := GetEnvironmentVariable(name, "", required)
	if output == "" {
		return defaultValue, nil
	}
	outputAsInt, err := strconv.ParseInt(output, 10, 64)
	if err != nil {
		return 0, err
	}
	if outputAsInt > math.MaxInt || outputAsInt < math.MinInt {
		return 0, fmt.Errorf("value out of range for int: %d", outputAsInt)
	}
	return int(outputAsInt), nil
}
