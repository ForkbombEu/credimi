#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

TMP_JSON="$(mktemp -t credimi-go-test-json-XXXXXX)"
trap 'rm -f "$TMP_JSON"' EXIT

TEST_TARGETS=("$@")
if [ "${#TEST_TARGETS[@]}" -eq 0 ]; then
	TEST_TARGETS=(./...)
fi

printf '\033[36mRunning unit tests (-race)...\033[0m\n'

set +e
go test -json -tags=unit -race -buildvcs "${TEST_TARGETS[@]}" 2>&1 | while IFS= read -r line; do
	printf '%s\n' "$line" >>"$TMP_JSON"

	if [[ "$line" == *'"Action":"'* ]] && [[ "$line" == *'"Test":"'* ]]; then
		action=""
		test_name=""
		package_name=""

		if [[ "$line" =~ \"Action\":\"([^\"]+)\" ]]; then
			action="${BASH_REMATCH[1]}"
		fi
		if [[ "$line" =~ \"Test\":\"([^\"\\]+)\" ]]; then
			test_name="${BASH_REMATCH[1]}"
		fi
		if [[ "$line" =~ \"Package\":\"([^\"\\]+)\" ]]; then
			package_name="${BASH_REMATCH[1]}"
		fi

		if [[ -n "$test_name" && "$test_name" != */* ]]; then
			case "$action" in
			pass)
				printf '\033[32m[PASS]\033[0m %s :: %s\n' "$package_name" "$test_name"
				;;
			fail)
				printf '\033[31m[FAIL]\033[0m %s :: %s\n' "$package_name" "$test_name"
				;;
			skip)
				printf '\033[33m[SKIP]\033[0m %s :: %s\n' "$package_name" "$test_name"
				;;
			esac
		fi
	fi
done
GO_TEST_STATUS=${PIPESTATUS[0]}
set -e

awk '
function extract(field, line, re, m) {
	re = "\"" field "\":\"(([^\"\\\\]|\\\\.)*)\""
	if (match(line, re, m)) {
		return m[1]
	}
	return ""
}

function top_test_name(test_name, parts) {
	split(test_name, parts, "/")
	return parts[1]
}

function test_key(pkg, test_name) {
	return pkg SUBSEP test_name
}

function normalize_file(file_path) {
	gsub(/\\\//, "/", file_path)
	sub(/^t\//, "/", file_path)
	return file_path
}

BEGIN {
	C_RESET = "\033[0m"
	C_GREEN = "\033[32m"
	C_RED = "\033[31m"
	C_YELLOW = "\033[33m"
	C_CYAN = "\033[36m"
}

{
	action = extract("Action", $0)
	pkg = extract("Package", $0)
	test_name = extract("Test", $0)
	output = extract("Output", $0)

	if (action == "run" && test_name != "" && index(test_name, "/") == 0) {
		key = test_key(pkg, test_name)
		if (!(key in seen_tests)) {
			seen_tests[key] = 1
			total_tests++
		}
		if (!(key in test_status)) {
			test_status[key] = "run"
		}
	}

	if ((action == "pass" || action == "fail" || action == "skip") &&
		test_name != "" && index(test_name, "/") == 0) {
		key = test_key(pkg, test_name)
		if (!(key in seen_tests)) {
			seen_tests[key] = 1
			total_tests++
		}
		test_status[key] = action
		if (action == "fail") {
			failed_tests[key] = 1
			packages_with_test_failures[pkg] = 1
		}
	}

	if (action == "output" && test_name != "") {
		top = top_test_name(test_name)
		key = test_key(pkg, top)
		if (!(key in test_file) &&
			match(output, /([A-Za-z0-9_./-]+_test\.go):[0-9]+:/, m)) {
			test_file[key] = m[1]
		}
	}

	if (action == "output" && test_name == "" && pkg != "") {
		if (!(pkg in package_file) &&
			match(output, /([A-Za-z0-9_./-]+\.go):[0-9]+:/, m)) {
			package_file[pkg] = m[1]
		}
	}

	if (match($0, /([A-Za-z0-9_./-]+_test\.go):[0-9]+/, raw_file_match)) {
		if (test_name != "") {
			top = top_test_name(test_name)
			key = test_key(pkg, top)
			if (!(key in test_file)) {
				test_file[key] = raw_file_match[1]
			}
		} else if (pkg != "" && !(pkg in package_file)) {
			package_file[pkg] = raw_file_match[1]
		}
	}

	if (action == "fail" && test_name == "" && pkg != "") {
		package_failures[pkg] = 1
	}
}

END {
	for (key in seen_tests) {
		if (test_status[key] == "pass") {
			passed_tests++
		} else if (test_status[key] == "skip") {
			skipped_tests++
		} else if (test_status[key] == "fail") {
			failed_test_count++
		}
	}

	for (pkg in package_failures) {
		if (!(pkg in packages_with_test_failures)) {
			package_only_failures++
		}
	}

	if (failed_test_count == 0 && package_only_failures == 0) {
		printf("%sPASS%s ", C_GREEN, C_RESET)
	} else {
		printf("%sFAIL%s ", C_RED, C_RESET)
	}

	printf("%sTop-level tests:%s %d/%d passed", C_CYAN, C_RESET, passed_tests, total_tests)
	if (skipped_tests > 0) {
		printf(" (%d skipped)", skipped_tests)
	}
	printf("\n")

	if (failed_test_count > 0) {
		printf("%sFailed tests:%s\n", C_RED, C_RESET)
		for (key in failed_tests) {
			split(key, parts, SUBSEP)
			pkg = parts[1]
			test_name = parts[2]
			if (key in test_file) {
				file = test_file[key]
			} else if (pkg in package_file) {
				file = package_file[pkg]
			} else {
				file = "<unknown>"
			}
			printf("  - %s (%s) [%s]\n", test_name, normalize_file(file), pkg)
		}
	}

	if (package_only_failures > 0) {
		printf("%sPackage failures (non-test errors):%s\n", C_YELLOW, C_RESET)
		for (pkg in package_failures) {
			if (pkg in packages_with_test_failures) {
				continue
			}
			file = (pkg in package_file) ? package_file[pkg] : "<unknown>"
			printf("  - %s [%s]\n", pkg, normalize_file(file))
		}
	}
}
' "$TMP_JSON"

exit "$GO_TEST_STATUS"
