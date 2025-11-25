// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
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

// SendMailActivityPayload is the input payload for the SendMailActivity.
type SendMailActivityPayload struct {
	Sender    string         `json:"sender,omitempty"   yaml:"sender,omitempty"`
	Recipient string         `json:"recipient"          yaml:"recipient"          validate:"required"`
	Subject   string         `json:"subject,omitempty"  yaml:"subject,omitempty"`
	Body      string         `json:"body,omitempty"     yaml:"body,omitempty"`
	Template  string         `json:"template,omitempty" yaml:"template,omitempty"`
	Data      map[string]any `json:"data,omitempty"     yaml:"data,omitempty"`
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
	if input.Config == nil {
		input.Config = make(map[string]string)
	}
	payload, err := workflowengine.DecodePayload[SendMailActivityPayload](input.Payload)
	if err != nil {
		return a.NewMissingOrInvalidPayloadError(err)
	}
	input.Config["smtp_host"] = utils.GetEnvironmentVariable("SMTP_HOST", "smtp.apps.forkbomb.eu")
	input.Config["smtp_port"] = utils.GetEnvironmentVariable("SMTP_PORT", "1025")
	payload.Sender = utils.GetEnvironmentVariable("MAIL_SENDER", "no-reply@credimi.io")
	input.Payload = payload
	return nil
}

func (a *SendMailActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[SendMailActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", payload.Sender)
	m.SetHeader("To", payload.Recipient)
	m.SetHeader("Subject", payload.Subject)
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	switch {
	case payload.Body != "" && payload.Template != "":
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'body' and 'template' cannot both be provided in payload",
				errCode.Description),
		)
	case payload.Body != "":
		m.SetBody("text/plain", payload.Body)
	case payload.Template != "" && payload.Data != nil:
		tmpl, err := template.New("email").Parse(payload.Template)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
			return workflowengine.ActivityResult{}, a.NewActivityError(
				errCode.Code,
				fmt.Sprintf("%s: %v", errCode.Description, err),
			)
		}
		var bodyBuffer bytes.Buffer
		if err := tmpl.Execute(&bodyBuffer, payload.Data); err != nil {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
			return workflowengine.ActivityResult{}, a.NewActivityError(
				errCode.Code,
				fmt.Sprintf("%s: %v", errCode.Description, err),
			)
		}
		m.SetBody("text/html", bodyBuffer.String())
	default:
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf(
				"%s: either 'body' or both 'template' and 'data' must be provided in payload",
				errCode.Description,
			),
		)
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
