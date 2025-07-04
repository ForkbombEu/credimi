// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"gopkg.in/gomail.v2"
)

// SendMailActivity is an activity that sends an email using SMTP.
type SendMailActivity struct {
	workflowengine.BaseActivity
}

func NewSendMailActivity() *SendMailActivity {
	return &SendMailActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Send an email",
		},
	}
}

// Name returns the name of the activity.
func (a *SendMailActivity) Name() string {
	return a.BaseActivity.Name
}

// Configure sets up the SMTP configuration for sending emails.
// It retrieves the SMTP host, port, and sender email from environment variables.
// If the environment variables are not set, it uses default values.
func (a *SendMailActivity) Configure(
	input *workflowengine.ActivityInput,
) error {
	input.Config["smtp_host"] = utils.GetEnvironmentVariable("SMTP_HOST", "smtp.apps.forkbomb.eu")
	input.Config["smtp_port"] = utils.GetEnvironmentVariable("SMTP_PORT", "1025")
	input.Config["sender"] = utils.GetEnvironmentVariable("MAIL_SENDER", "no-reply@credimi.io")
	return nil
}

// Execute sends an email using the provided SMTP configuration and payload.
func (a *SendMailActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult

	m := gomail.NewMessage()
	m.SetHeader("From", input.Config["sender"])
	m.SetHeader("To", input.Config["recipient"])
	m.SetHeader("Subject", input.Payload["subject"].(string))
	m.SetBody("text/html", input.Payload["body"].(string))

	// Attach any files if necessary
	attachments, ok := input.Payload["attachments"].(map[string][]byte)
	if ok {
		for filename, attachedBytes := range attachments {
			attached := gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(attachedBytes)
				return err
			})
			m.Attach(filename, attached)
		}
	}

	SMTPPort, err := strconv.Atoi(input.Config["smtp_port"])
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'SMTP_PORT environment variable not an integer'", errCode.Description),
			input.Config["smtp_port"],
		)
	}

	d := gomail.NewDialer(
		input.Config["smtp_host"],
		SMTPPort,
		utils.GetEnvironmentVariable("MAIL_USERNAME"),
		utils.GetEnvironmentVariable("MAIL_PASSWORD"),
	)

	if err := d.DialAndSend(m); err != nil {
		errCode := errorcodes.Codes[errorcodes.EmailSendFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
		)
	}

	result.Output = "Email sent successfully"
	return result, nil
}
