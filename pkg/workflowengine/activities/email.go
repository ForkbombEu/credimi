// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// SendMailActivity is an activity that sends an email using SMTP.
type SendMailActivity struct{}

// Name returns the name of the activity.
func (SendMailActivity) Name() string {
	return "Send an email"
}

// Configure sets up the SMTP configuration for sending emails.
// It retrieves the SMTP host, port, and sender email from environment variables.
// If the environment variables are not set, it uses default values.
func (a *SendMailActivity) Configure(
	_ context.Context,
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
		return workflowengine.Fail(&result, "SMTP_PORT environment variable not an integer")
	}

	d := gomail.NewDialer(
		input.Config["smtp_host"],
		SMTPPort,
		os.Getenv("MAIL_USERNAME"),
		os.Getenv("MAIL_PASSWORD"),
	)

	if err := d.DialAndSend(m); err != nil {
		return workflowengine.Fail(&result, fmt.Sprintf("failed to send email: %v", err))
	}

	result.Output = "Email sent successfully"
	return result, nil
}
