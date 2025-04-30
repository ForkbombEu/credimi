// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"os"
	"testing"

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
	activity := &SendMailActivity{}
	input := &workflowengine.ActivityInput{
		Config: make(map[string]string),
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
			require.Equal(t, "sender@example.com", input.Config["sender"])
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
		name           string
		input          workflowengine.ActivityInput
		expectedOutput string
		expectedErr    string
	}{
		{
			name: "Success - email sent successfully",
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"smtp_host": "localhost", // point to the smtpmock server
					"smtp_port": "2525",
					"sender":    "sender@example.com",
					"recipient": "recipient@example.com",
				},
				Payload: map[string]interface{}{
					"subject": "Test Email",
					"body":    "<html><body>Test email body</body></html>",
				},
			},
			expectedOutput: "Email sent successfully",
			expectedErr:    "",
		},
		{
			name: "Failure - missing recipient email",
			input: workflowengine.ActivityInput{
				Config: map[string]string{
					"smtp_host": "localhost",
					"smtp_port": "2525",
					"sender":    "sender@example.com",
				},
				Payload: map[string]interface{}{
					"subject": "Test Email",
					"body":    "<html><body>Test email body</body></html>",
				},
			},
			expectedOutput: "Email sending failed",
			expectedErr:    "no address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result workflowengine.ActivityResult
			future, err := env.ExecuteActivity(activity.Execute, tt.input)
			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				future.Get(&result)
				require.Equal(t, tt.expectedOutput, result.Output)
			}
		})
	}
}
