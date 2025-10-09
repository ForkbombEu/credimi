package activities

import (
	"fmt"
	"os"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type CheckFileExistsActivity struct {
	workflowengine.BaseActivity
}

func NewCheckFileExistsActivity() *CheckFileExistsActivity {
	return &CheckFileExistsActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Check if a file exists",
		},
	}
}

func (a *CheckFileExistsActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *CheckFileExistsActivity) Execute(input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
	path, ok := input.Payload["path"].(string)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'path'", errCode.Description),
		)
	}
	_, err := os.Stat(path)
	exists := err == nil
	return workflowengine.ActivityResult{
		Output: exists,
	}, nil
}
