// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"os"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	smtpmock "github.com/mocktools/go-smtp-mock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"gopkg.in/gomail.v2"
)

type MockDialer struct {
	mock.Mock
}

func (m *MockDialer) DialAndSend(msg *gomail.Message) error {
	args := m.Called(msg)
	return args.Error(0)
}

func TestSendMailActivity_Configure(t *testing.T) {
	activity := NewSendMailActivity()
	input := &workflowengine.ActivityInput{
		Config: make(map[string]string),
		Payload: SendMailActivityPayload{
			Recipient: "test@example.com",
		},
	}
	tests := []struct {
		name     string
		setupEnv func()
	}{
		{
			name: "Success - valid environment variables",
			setupEnv: func() {
				os.Setenv("SMTP_HOST", "smtp.example.com")
				os.Setenv("SMTP_PORT", "587")
				os.Setenv("MAIL_SENDER", "sender@example.com")
			},
		},
	}

	// Run each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			err := activity.Configure(input)

			require.NoError(t, err)
			require.Equal(t, "smtp.example.com", input.Config["smtp_host"])
			require.Equal(t, "587", input.Config["smtp_port"])
			payload, err := workflowengine.DecodePayload[SendMailActivityPayload](input.Payload)
			require.NoError(t, err)
			require.Equal(t, "sender@example.com", payload.Sender)
		})
	}
}

func TestSendMailActivity_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()

	// Start mock SMTP server on port 2525
	mockServer := smtpmock.New(smtpmock.ConfigurationAttr{
		PortNumber:  2525,
		LogToStdout: false,
	})
	if err := mockServer.Start(); err != nil {
		t.Fatalf("failed to start mock SMTP server: %v", err)
	}
	defer mockServer.Stop()

	// Use the real activity
	activity := &SendMailActivity{}
	env.RegisterActivity(activity.Execute)

	tests := []struct {
		name            string
		input           workflowengine.ActivityInput
		expectedOutput  string
		expectedErr     bool
		expectedErrCode errorcodes.Code
	}{
		{
			name: "Success - email sent successfully",
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"smtp_host": "localhost",
					"smtp_port": "2525",
				},
				Payload: SendMailActivityPayload{
					Sender:    "sender@example.com",
					Recipient: "recipient@example.com",
					Subject:   "Test Email",
					Body:      "<html><body>Test email body</body></html>",
				},
			},
			expectedOutput: "Email sent successfully",
		},
		{
			name: "Failure - missing recipient email",
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"smtp_host": "localhost",
					"smtp_port": "2525",
				},
				Payload: SendMailActivityPayload{
					Sender:  "sender@example.com",
					Subject: "Test Email",
					Body:    "<html><body>Test email body</body></html>",
				},
			},
			expectedOutput:  "Email sending failed",
			expectedErr:     true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result workflowengine.ActivityResult
			future, err := env.ExecuteActivity(activity.Execute, tt.input)
			if tt.expectedErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrCode.Code)
				require.Contains(t, err.Error(), tt.expectedErrCode.Description)
			} else {
				require.NoError(t, err)
				future.Get(&result)
				require.Equal(t, tt.expectedOutput, result.Output)
			}
		})
	}
}
