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

	"github.com/forkbombeu/didimo/pkg/utils"
	workflowengine "github.com/forkbombeu/didimo/pkg/workflow_engine"
)

type SendMailActivity struct{}

func (SendMailActivity) Name() string {
	return "Send an email"
}

func (a *SendMailActivity) Configure(
	ctx context.Context,
	input *workflowengine.ActivityInput,
) error {
	input.Config["smtp_host"] = utils.GetEnvironmentVariable("SMTP_HOST", "smtp.apps.forkbomb.eu")
	input.Config["smtp_port"] = utils.GetEnvironmentVariable("SMTP_PORT", "1025")
	input.Config["sender"] = utils.GetEnvironmentVariable("MAIL_SENDER", "no-reply@credimi.io")
	return nil
}

func (a *SendMailActivity) Execute(
	ctx context.Context,
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
