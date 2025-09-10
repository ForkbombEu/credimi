// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type MaestroFlowActivity struct {
	workflowengine.BaseActivity
}

func NewMaestroFlowActivity() *MaestroFlowActivity {
	return &MaestroFlowActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Run a Maestro flow",
		},
	}
}
func (a *MaestroFlowActivity) Name() string {
	return a.BaseActivity.Name
}
func (a *MaestroFlowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult

	yamlContent, ok := input.Payload["yaml"].(string)

	if !ok || yamlContent == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'yaml", errCode.Description),
		)
	}

	apk, ok := input.Payload["apk"].(string)
	if !ok || apk == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'apk", errCode.Description),
		)
	}

	cmd := exec.CommandContext(ctx, "adb", "install", apk)
	err := cmd.Run()
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf(errCode.Description+": %v", err),
		)
	}

	tmpFile, err := os.CreateTemp("", "action.yaml")
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.TempFileCreationFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf(errCode.Description+": %v", err),
		)
	}
	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		errCode := errorcodes.Codes[errorcodes.TempFileCreationFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf(errCode.Description+": %v", err),
		)
	}
	// defer os.Remove(tmpFile.Name())

	binDir := utils.GetEnvironmentVariable("BIN", ".bin")
	exexcutableDir := filepath.Join(binDir, "maestro", "bin")
	binName := "maestro"
	binPath := filepath.Join(exexcutableDir, binName)
	args := []string{"test", tmpFile.Name(), "--output", "video.mp4"}

	cmd = exec.CommandContext(ctx, binPath, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf(errCode.Description+": %v", err),
			stderrStr,
		)
	}

	result.Output = stdoutStr

	return result, nil
}
