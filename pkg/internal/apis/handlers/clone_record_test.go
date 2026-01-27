// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"io"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

func getTestOrgID() (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := `name="userA's organization"`

	record, err := app.FindFirstRecordByFilter("organizations", filter)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func setupApp(orgID string) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(app)
		CloneRecord.Add(app)

		coll, _ := app.FindCollectionByNameOrId("credential_issuers")
		issuerRecord := core.NewRecord(coll)
		issuerRecord.Set("id", "tikklnj1uh32237")
		issuerRecord.Set("name", "test issuer")
		issuerRecord.Set("url", "https://test-issuer.example.com")
		issuerRecord.Set("owner", orgID)
		require.NoError(t, app.Save(issuerRecord))

		credColl, _ := app.FindCollectionByNameOrId("credentials")
		r := core.NewRecord(credColl)
		r.Set("id", "crede1234567890")
		r.Set("owner", orgID)
		r.Set("name", "test credential")
		r.Set("credential_issuer", issuerRecord.Id)
		require.NoError(t, app.Save(r))
		return app
	}
}

func TestGetCloneRecord(t *testing.T) {
	orgID, err := getTestOrgID()
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)
	userRecordNotAuth, err := getUserRecordFromName("userB")
	require.NoError(t, err)
	tokenUserNotAuth, err := userRecordNotAuth.NewAuthToken()
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "get clone-record success",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "credentials"
			}`),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"cloned_record"`,
				`"message"`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone with invalid JSON",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "credentials",`), // JSON malformato
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Invalid JSON."`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone with empty collection",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": ""
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Id and collection are required."`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone with not supported collection",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "coll"
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Collection 'coll' not supported for cloning."`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "record not found in collection",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567891",
				"collection": "credentials"
			}`),
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"reason":"Record 'crede1234567891' not found in collection 'credentials'"`,
				`"message":"sql: no rows in result set"`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone without authentication",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "credentials"
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Authentication required."`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone with unauthorized user",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": tokenUserNotAuth,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "credentials"
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Not authorized for this organization."`,
			},
			TestAppFactory: setupApp(orgID),
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func setupTestApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	return app
}

func TestCloneFiles_NoFileFieldValues(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)

	originalRecord := core.NewRecord(coll)
	newRecord := core.NewRecord(coll)

	err = cloneFiles(app, originalRecord, newRecord, map[string]interface{}{})
	require.NoError(t, err, "Should succeed with empty file field values")
}

func TestCloneFiles_SingleFile(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()
	own, _ := getTestOrgID()
	coll, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)

	collcred, _ := app.FindCollectionByNameOrId("credential_issuers")
	issuerRecord := core.NewRecord(collcred)
	issuerRecord.Set("id", "tikklnj1uh32237")
	issuerRecord.Set("name", "test issuer")
	issuerRecord.Set("url", "https://test-issuer.example.com")
	issuerRecord.Set("owner", own)
	require.NoError(t, app.Save(issuerRecord))

	originalRecord := core.NewRecord(coll)
	originalRecord.Set("id", "original1234567")
	originalRecord.Set("owner", own)
	originalRecord.Set("name", "original record")
	originalRecord.Set("canonified_name", "original_record")
	originalRecord.Set("credential_issuer", issuerRecord.Id)

	testData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43, 0x00, 0xFF,
		0xC9, 0x00, 0x0B, 0x08, 0x00, 0x01, 0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF,
		0xCC, 0x00, 0x06, 0x00, 0x10, 0x10, 0x05, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01,
		0x00, 0x00, 0x3F, 0x00, 0xD2, 0xCF, 0x20, 0xFF, 0xD9,
	}

	testFile, err := filesystem.NewFileFromBytes(testData, "testimage.jpg")
	require.NoError(t, err)

	originalRecord.Set("logo", testFile)
	err = app.Save(originalRecord)
	require.NoError(t, err)

	originalFilename := originalRecord.GetString("logo")
	require.NotEmpty(t, originalFilename, "Original record should have the file")

	newRecord := core.NewRecord(coll)
	newRecord.Set("id", "new456789012345")
	newRecord.Set("owner", own)
	newRecord.Set("name", "new record copy2345")
	newRecord.Set("canonified_name", "new_record_copy2345")
	newRecord.Set("credential_issuer", issuerRecord.Id)

	fileFieldValues := map[string]interface{}{
		"logo": originalFilename,
	}

	err = cloneFiles(app, originalRecord, newRecord, fileFieldValues)
	require.NoError(t, err, "Should clone single file successfully")

	clonedFilename := newRecord.GetString("logo")
	require.NotEmpty(t, clonedFilename, "New record should have the cloned file")

	fsys, err := app.NewFilesystem()
	require.NoError(t, err)
	defer fsys.Close()

	clonedFileKey := newRecord.BaseFilesPath() + "/" + clonedFilename
	r, err := fsys.GetFile(clonedFileKey)
	require.NoError(t, err)
	defer r.Close()

	clonedData, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, testData, clonedData, "Cloned file should have same content")

}

func TestCloneFiles_FileNotFound(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()
	own, _ := getTestOrgID()
	coll, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)

	collcred, _ := app.FindCollectionByNameOrId("credential_issuers")
	issuerRecord := core.NewRecord(collcred)
	issuerRecord.Set("id", "tikklnj1uh32237")
	issuerRecord.Set("name", "test issuer notfound")
	issuerRecord.Set("url", "https://test-issuer-notfound.example.com")
	issuerRecord.Set("owner", own)
	require.NoError(t, app.Save(issuerRecord))

	originalRecord := core.NewRecord(coll)
	originalRecord.Set("id", "original1234567")
	originalRecord.Set("owner", own)
	originalRecord.Set("name", "original notfound test")
	originalRecord.Set("canonified_name", "original_notfound_test")
	originalRecord.Set("credential_issuer", issuerRecord.Id)
	require.NoError(t, app.Save(originalRecord))

	newRecord := core.NewRecord(coll)
	newRecord.Set("id", "new456789012345")
	newRecord.Set("owner", own)
	newRecord.Set("name", "new notfound test")
	newRecord.Set("canonified_name", "new_notfound_test")
	newRecord.Set("credential_issuer", issuerRecord.Id)

	fileFieldValues := map[string]interface{}{
		"logo": "non_existent_file_12345.jpg",
	}

	err = cloneFiles(app, originalRecord, newRecord, fileFieldValues)
	require.NoError(t, err, "Should not error when file not found")

	logoValue := newRecord.Get("logo")
	require.Empty(t, logoValue, "Field should be empty when file not found")

}

func TestCloneFiles_MoreFiles(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()
	own, _ := getTestOrgID()
	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	collpip, _ := app.FindCollectionByNameOrId("pipelines")
	pipRecord := core.NewRecord(collpip)
	pipRecord.Set("id", "tikklnj1uh32237")
	pipRecord.Set("name", "test pipeline")
	pipRecord.Set("canonified_name", "test_pipeline")
	pipRecord.Set("description", "my pipeline")
	pipRecord.Set("steps", "{[]}")
	pipRecord.Set("yaml", "yaml")
	pipRecord.Set("owner", own)
	require.NoError(t, app.Save(pipRecord))

	originalRecord := core.NewRecord(coll)
	originalRecord.Set("id", "original1234567")
	originalRecord.Set("owner", own)
	originalRecord.Set("pipeline", pipRecord.Id)
	originalRecord.Set("workflow_id", "123")
	originalRecord.Set("run_id", "456")
	originalRecord.Set("canonified_identifier", "123-456")

	testData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43, 0x00, 0xFF,
		0xC9, 0x00, 0x0B, 0x08, 0x00, 0x01, 0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF,
		0xCC, 0x00, 0x06, 0x00, 0x10, 0x10, 0x05, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01,
		0x00, 0x00, 0x3F, 0x00, 0xD2, 0xCF, 0x20, 0xFF, 0xD9,
	}

	testFile1, err := filesystem.NewFileFromBytes(testData, "screenshot1.jpg")
	require.NoError(t, err)

	testFile2, err := filesystem.NewFileFromBytes(testData, "screenshot2.png")
	require.NoError(t, err)

	originalRecord.Set("screenshots", []*filesystem.File{testFile1, testFile2})
	err = app.Save(originalRecord)
	require.NoError(t, err)

	screenshotsValue := originalRecord.Get("screenshots")
	require.NotNil(t, screenshotsValue, "screenshots should not be nil")

	newRecord := core.NewRecord(coll)
	newRecord.Set("id", "new456789012345")
	newRecord.Set("owner", own)
	newRecord.Set("workflow_id", "123")
	newRecord.Set("run_id", "456")
	newRecord.Set("pipeline", pipRecord.Id)
	originalRecord.Set("canonified_identifier", "123-456-1")

	fileFieldValues := map[string]interface{}{
		"screenshots": screenshotsValue,
	}

	err = cloneFiles(app, originalRecord, newRecord, fileFieldValues)
	require.NoError(t, err, "Should clone multiple files successfully")

	clonedValue := newRecord.Get("screenshots")
	require.NotNil(t, clonedValue, "Cloned screenshots should not be nil")

	fsys, err := app.NewFilesystem()
	require.NoError(t, err)
	defer fsys.Close()

	var clonedFilenames []string

	switch v := clonedValue.(type) {
	case []string:
		clonedFilenames = v
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				clonedFilenames = append(clonedFilenames, str)
			}
		}
	default:
		t.Fatalf("Unexpected type for cloned screenshots: %T", v)
	}

	require.Len(t, clonedFilenames, 2, "Should have cloned 2 files")

	for i, filename := range clonedFilenames {
		filePath := newRecord.BaseFilesPath() + "/" + filename
		_, err := fsys.GetFile(filePath)
		require.NoError(t, err, "Cloned file %d should exist: %s", i+1, filename)
	}

}

func setupAppWithPipelines(orgID string) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(app)
		CloneRecord.Add(app)
		
		coll, _ := app.FindCollectionByNameOrId("pipelines")
		pipelineRecord := core.NewRecord(coll)
		pipelineRecord.Set("id", "tikklnj1uh32237")
		pipelineRecord.Set("name", "test pipeline")
		pipelineRecord.Set("canonified_name", "test_pipeline")
		pipelineRecord.Set("description", "A test pipeline for cloning")
		pipelineRecord.Set("steps", `[{"name": "step1", "type": "http"}]`)
		pipelineRecord.Set("yaml", "version: 1.0")
		pipelineRecord.Set("owner", orgID) 
		pipelineRecord.Set("published", false)
		require.NoError(t, app.Save(pipelineRecord))
		
		return app
	}
}

func TestCloneRecord_WithBeforeSave(t *testing.T) {
	orgID, err := getTestOrgID()
	require.NoError(t, err)
	
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	userRecordNotAuth, err := getUserRecordFromName("userB")
	require.NoError(t, err)
	tokenUserNotAuth, err := userRecordNotAuth.NewAuthToken()
	require.NoError(t, err)
	
	scenarios := []tests.ApiScenario{
		{
			Name:   "clone pipeline success with BeforeSave",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "tikklnj1uh32237",
				"collection": "pipelines"
			}`),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"cloned_record"`,
				`"message":"Record cloned from 'pipelines'"`,
				`"owner":"` + orgID + `"`, 
				`"published":false`, 
			},
			TestAppFactory: setupAppWithPipelines(orgID),
		},
		{
			Name:   "clone pipeline with unauthorized user",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": tokenUserNotAuth,
			},
			Body: strings.NewReader(`{
				"id": "tikklnj1uh32237",
				"collection": "pipelines"
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Not authorized for this organization."`,
			},
			TestAppFactory: setupAppWithPipelines(orgID),
		},
		{
			Name:   "clone pipeline with no authentication",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: strings.NewReader(`{
				"id": "tikklnj1uh32237",
				"collection": "pipelines"
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Authentication required."`,
			},
			TestAppFactory: setupAppWithPipelines(orgID),
		},
		{
			Name:   "clone published pipeline with different user",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": tokenUserNotAuth,
			},
			Body: strings.NewReader(`{
				"id": "tikklnj1uh32237",
				"collection": "pipelines"
			}`),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"cloned_record"`,
				`"message":"Record cloned from 'pipelines'"`,
				`"owner":"3u4982xn6ah0433"`, 
				`"published":false`, 
			},	
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app, err := tests.NewTestApp(testDataDir)
				require.NoError(t, err)
				canonify.RegisterCanonifyHooks(app)
				CloneRecord.Add(app)
				
				coll, _ := app.FindCollectionByNameOrId("pipelines")
				pipelineRecord := core.NewRecord(coll)
				pipelineRecord.Set("id", "tikklnj1uh32237")
				pipelineRecord.Set("name", "Published Pipeline")
				pipelineRecord.Set("canonified_name", "published_pipeline")
				pipelineRecord.Set("description", "A published pipeline")
				pipelineRecord.Set("steps", `[{"name": "step1", "type": "http"}]`)
				pipelineRecord.Set("yaml", "version: 1.0")
				pipelineRecord.Set("owner", orgID)
				pipelineRecord.Set("published", true) 
				require.NoError(t, app.Save(pipelineRecord))
				
				return app
			},
		},
	}
	
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
