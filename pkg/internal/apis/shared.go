// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *UnitTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

const testDataDir = "../../../test_pb_data"

func generateToken(collectionNameOrID string, email string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	record, err := app.FindAuthRecordByEmail(collectionNameOrID, email)
	if err != nil {
		return "", err
	}

	return record.NewAuthToken()
}

func isSuperUser(app core.App, user *core.Record) bool {
	superUserRecord, err := app.FindFirstRecordByFilter(
		"_superusers",
		"id ={:id}",
		dbx.Params{"id": user.Id},
	)
	if err != nil {
		return false
	}
	if superUserRecord != nil {
		return true
	}
	return false
}
