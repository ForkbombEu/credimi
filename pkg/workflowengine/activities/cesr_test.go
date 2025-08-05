// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/ForkbombEu/et-tu-cesr/cesr"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"golang.org/x/sys/unix"
)

const (
	ValidCESR   = `{"v":"KERI10JSON000249_","t":"dip","d":"Ez6QKIKLzrGqpq4v9Bj908pQanoRKwOgBXjPW-w-P_8Q","i":"Ez6QKIKLzrGqpq4v9Bj908pQanoRKwOgBXjPW-w-P_8Q","s":"0","kt":"2","k":["DQGnP_wcQSoIYd9U9rmLw75lZ__9UYy6LVekVjvdeDqw","DeVruwFGOOPVugouR8afsKPcw1bxt674uMCvdLaub7Do"],"nt":"2","n":["EcHq2C5-Gc5NeVOb-YYKbr8gh-Z6VqMbryO5XFsNRhb0","E3JYJXrYXFm6HCSZNCDlbQrA8FiLA3LiFVOFy0Aix2Ww"],"bt":"3","b":["Boq71an-vhU6DtlZzzJF7yIqbQxb56rcxeB0LppxeDOA","BHGK9Gem8PdiZ7PZ9WcIwxM7YnGaztYA65X3o5_RxFa8","B4tbPLI_TEze0pzAA-X-gewpdg22yfzN8CdKKIF5wETM"],"c":[],"a":[],"di":"EC1m0ZF6ez1xoM8-jQsIbT5I3GpYnX4Zzh4om8_V1bnU"}-AAA-random{"v":"ACDC10JSON000197_","d":"Eb9r5x3NPd4iwvPnOiE2B-x7yNGTrM1Bxg9GoysCixwU","i":"Ez6QKIKLzrGqpq4v9Bj908pQanoRKwOgBXjPW-w-P_8Q","ri":"EiySMSMLRrpt9AD_b34VEtUD6tDO0yaGoWCwB1iyJOgY","s":"EWCeT9zTxaZkaC_3-amV2JtG6oUxNA36sCC0P5MI7Buw","a":{"d":"EfxwhLev_vcdraCjayPVfb0wmZtCt8_ARt_FvdKtNu2Q","dt":"2022-06-21T13:46:09.308721+00:00","i":"E0m0vlIMbPVbNVfPTH3NcLW0iagpyke_4OVZN7YNFLkE","LEI":"529900T8BM49AURSDO55"}}-AAA-random`
	InvalidCESR = `{"v":"KERI10JSON000018_"}{"i":"abc"}`
	ValidCRED   = ` [{"AttachBytes":11,"KED":{"a":{"LEI":"529900T8BM49AURSDO55","d":"EfxwhLev_vcdraCjayPVfb0wmZtCt8_ARt_FvdKtNu2Q","dt":"2022-06-21T13:46:09.308721+00:00","i":"E0m0vlIMbPVbNVfPTH3NcLW0iagpyke_4OVZN7YNFLkE"},"d":"Eb9r5x3NPd4iwvPnOiE2B-x7yNGTrM1Bxg9GoysCixwU","i":"Ez6QKIKLzrGqpq4v9Bj908pQanoRKwOgBXjPW-w-P_8Q","ri":"EiySMSMLRrpt9AD_b34VEtUD6tDO0yaGoWCwB1iyJOgY","s":"EWCeT9zTxaZkaC_3-amV2JtG6oUxNA36sCC0P5MI7Buw","v":"ACDC10JSON000197_"}}]`
	InvalidCRED = `[{"AttachBytes":11,"KED":{"a":{"ERR":"529900T8BM49AURSDO55","d":"EfxwhLev_vcdraCjayPVfb0wmZtCt8_ARt_FvdKtNu2Q","dt":"2022-06-21T13:46:09.308721+00:00","i":"E0m0vlIMbPVbNVfPTH3NcLW0iagpyke_4OVZN7YNFLkE"},"d":"Eb9r5x3NPd4iwvPnOiE2B-x7yNGTrM1Bxg9GoysCixwU","i":"Ez6QKIKLzrGqpq4v9Bj908pQanoRKwOgBXjPW-w-P_8Q","ri":"EiySMSMLRrpt9AD_b34VEtUD6tDO0yaGoWCwB1iyJOgY","s":"EWCeT9zTxaZkaC_3-amV2JtG6oUxNA36sCC0P5MI7Buw","v":"ACDC10JSON000197_"}}]`
)

func TestCESRParsing_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()

	activity := NewCESRParsingActivity()
	env.RegisterActivity(activity.Execute)

	tests := []struct {
		name            string
		rawCESR         string
		expectErr       bool
		expectedErrCode errorcodes.Code
		expectValue     []cesr.Event
	}{
		{
			name:    "Success - valid CESR",
			rawCESR: ValidCESR,
			expectValue: []cesr.Event{
				{
					KED: map[string]any{
						"a": []any{},
						"b": []any{
							"Boq71an-vhU6DtlZzzJF7yIqbQxb56rcxeB0LppxeDOA",
							"BHGK9Gem8PdiZ7PZ9WcIwxM7YnGaztYA65X3o5_RxFa8",
							"B4tbPLI_TEze0pzAA-X-gewpdg22yfzN8CdKKIF5wETM",
						},
						"bt": "3",
						"c":  []any{},
						"d":  "Ez6QKIKLzrGqpq4v9Bj908pQanoRKwOgBXjPW-w-P_8Q",
						"di": "EC1m0ZF6ez1xoM8-jQsIbT5I3GpYnX4Zzh4om8_V1bnU",
						"i":  "Ez6QKIKLzrGqpq4v9Bj908pQanoRKwOgBXjPW-w-P_8Q",
						"k": []any{
							"DQGnP_wcQSoIYd9U9rmLw75lZ__9UYy6LVekVjvdeDqw",
							"DeVruwFGOOPVugouR8afsKPcw1bxt674uMCvdLaub7Do",
						},
						"kt": "2",
						"n": []any{
							"EcHq2C5-Gc5NeVOb-YYKbr8gh-Z6VqMbryO5XFsNRhb0",
							"E3JYJXrYXFm6HCSZNCDlbQrA8FiLA3LiFVOFy0Aix2Ww",
						},
						"nt": "2",
						"s":  "0",
						"t":  "dip",
						"v":  "KERI10JSON000249_",
					},
					AttachBytes: 11,
				},
				{
					KED: map[string]any{
						"a": map[string]any{
							"LEI": "529900T8BM49AURSDO55",
							"d":   "EfxwhLev_vcdraCjayPVfb0wmZtCt8_ARt_FvdKtNu2Q",
							"dt":  "2022-06-21T13:46:09.308721+00:00",
							"i":   "E0m0vlIMbPVbNVfPTH3NcLW0iagpyke_4OVZN7YNFLkE",
						},
						"d":  "Eb9r5x3NPd4iwvPnOiE2B-x7yNGTrM1Bxg9GoysCixwU",
						"i":  "Ez6QKIKLzrGqpq4v9Bj908pQanoRKwOgBXjPW-w-P_8Q",
						"ri": "EiySMSMLRrpt9AD_b34VEtUD6tDO0yaGoWCwB1iyJOgY",
						"s":  "EWCeT9zTxaZkaC_3-amV2JtG6oUxNA36sCC0P5MI7Buw",
						"v":  "ACDC10JSON000197_",
					},
					AttachBytes: 11,
				},
			},
		},
		{
			name:            "Failure - invalid CESR",
			rawCESR:         InvalidCESR,
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.CESRParsingError],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]any{
				"rawCESR": tt.rawCESR,
			}

			input := workflowengine.ActivityInput{
				Payload: payload,
			}

			future, err := env.ExecuteActivity(activity.Execute, input)

			if tt.expectErr {
				require.Error(t, err)
				if tt.expectedErrCode != (errorcodes.Code{}) {
					require.Contains(t, err.Error(), tt.expectedErrCode.Code)
					require.Contains(t, err.Error(), tt.expectedErrCode.Description)
				}
			} else {
				require.NoError(t, err)
				var result workflowengine.ActivityResult
				err := future.Get(&result)
				require.NoError(t, err)
				jsonBytes, err := json.Marshal(result.Output)
				require.NoError(t, err)
				var actual []cesr.Event
				err = json.Unmarshal(jsonBytes, &actual)
				require.NoError(t, err)
				require.Equal(t, tt.expectValue, actual)
			}
		})
	}
}
func TestCESRValidate_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	activity := &CESRValidateActivity{}
	env.RegisterActivity(activity.Execute)

	tmpBinDir := t.TempDir()
	binPath := fmt.Sprintf("%s/et-tu-cesr", tmpBinDir)

	// Determine the platform and architecture
	OS := runtime.GOOS
	var utsname unix.Utsname
	unix.Uname(&utsname)
	arch := string(bytes.Trim(utsname.Machine[:], "\x00"))

	// Construct the binary download URL
	url := fmt.Sprintf(
		"https://github.com/ForkbombEu/et-tu-cesr/releases/latest/download/et-tu-cesr-%s-%s",
		OS,
		arch,
	)

	// Download the binary from GitHub
	cmd := exec.Command("wget", url, "-O", binPath)
	cmd.Dir = tmpBinDir // Set working directory to the temporary binary directory

	t.Logf("Downloading binary from: %s", url)
	err := cmd.Run()
	require.NoError(t, err, "Failed to download binary")

	// Make the binary executable
	err = os.Chmod(binPath, 0755)
	require.NoError(t, err, "Failed to make binary executable")

	// Set environment variable to point to the binary directory
	os.Setenv("BIN", tmpBinDir)

	tests := []struct {
		name             string
		payload          map[string]interface{}
		expectedError    bool
		expectedErrorMsg errorcodes.Code
		expectedOutput   string
	}{
		{
			name: "Success - validation correct",
			payload: map[string]interface{}{
				"events": ValidCRED,
			},
			expectedOutput: "1 credential bodies valid",
		},
		{
			name: "Failure - validation fails",
			payload: map[string]any{
				"events": InvalidCRED,
			},
			expectedError:    true,
			expectedErrorMsg: errorcodes.Codes[errorcodes.CommandExecutionFailed],
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := workflowengine.ActivityInput{
				Payload: tc.payload,
			}
			var result workflowengine.ActivityResult
			future, err := env.ExecuteActivity(activity.Execute, input)
			if tc.expectedError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErrorMsg.Code)
				require.Contains(t, err.Error(), tc.expectedErrorMsg.Description)
			} else {
				require.NoError(t, err)
				future.Get(&result)
				if tc.expectedOutput != "" {
					require.Contains(t, result.Output.(string), tc.expectedOutput)
				}
			}
		})
	}
}
