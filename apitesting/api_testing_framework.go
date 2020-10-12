// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package apitesting contains integration testing framework functions
package apitesting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpexpect "github.com/gavv/httpexpect/v2"
)

// APITest is a structure used to define some environmental aspects of a test run (like Setup/Teardown functions)
// It also holds some internal context.
// Usage: using httptest as API endpoint
// 		apitest := apitesting.APITest{
// 		GetHTTPHandler: func(ctx context.Context, t *testing.T) http.Handler {
// 			return initializer.Init()    // assumes Init() returns http.Hanlder
// 		},
// Usage: using actual HTTP endpoint
// 		apitest := apitesting.APITest{
//					APIEndpointURL: "http://some.host.com:8000"
//
//   Then you created test cases via the ApiTestCase structure:
//		tests := []apitesting.APITestCase{
//		{
//			Name:             "Create a product with invalid sstoken",
//			ReqType:          "POST",
// 			ResourcePath:     func(ctx context.Context, t *testing.T) string { return "/someurl" },
//			PopulateHeaders:  func(ctx context.Context, t *testing.T) map[string]string { return map[string]string{"Authorization": "a_token"} },
//          PopulateJSON:     func(ctx context.Context, t *testing.T) interface{} { return models.testdata },
//			ExpHTTPStatus:    http.StatusOK,
//          BeforeRequest:    func(ctx context.Context, t *testing.T, e *httpexpect.Expect) { // do something before the request is sent },
//			ValidateResponse: func(t *testing.T, resp *httpexpect.Response) context.Context { // validate the response object },
//          AfterValidate:    func(ctx context.Context, t *testing.T, e *httpexpect.Expect) { // do something just before this testcase is torn down },
//		}
//   Then you run the test cases:
//          apitest.RunTests(t, tests)
//
type APITest struct {
	// GetHTTPHandler returns an http.Handler (e.g. *gin.Engine) which will be bootstrapped with httptest
	// Only one of GetHTTPHandler and APIEndpointURL should be provided
	GetHTTPHandler func(ctx context.Context, t *testing.T) http.Handler
	// APIEndpointURL is the HTTP endpoint to send the HTTP calls to
	// Only one of GetHTTPHandler and APIEndpointURL should be provided
	APIEndpointURL string
	//
	// A function that will run once before any tests execute.
	SetupTest func(ctx context.Context, t *testing.T)
	// A function that will run once after all tests have run (optional)
	TeardownTest func(ctx context.Context, t *testing.T)
	// A function that will run before each test case (i.e. before each element in the []ApiTestCase passed to RunTests)
	SetupTestCase func(ctx context.Context, t *testing.T, tc APITestCase)
	// A function that will run after each test case
	TeardownTestCase func(ctx context.Context, t *testing.T, tc APITestCase)
	// A Context object that can be used to maintain context between tests/test cases
	Ctx    context.Context
	server *httptest.Server
}

// APITestCase is a structure used to define the specific test cases to be run via RunTests.
type APITestCase struct {
	// Description of the test case
	Name string
	// HTTP Request type (e.g. "POST", "GET")
	ReqType string
	// A function that returns the path to the resource to call (e.g. /v1/product or /v1/product/<uuid> etc)
	ResourcePath func(ctx context.Context, t *testing.T) string
	// A function that will be run immediately before the testcase request is sent
	BeforeRequest func(ctx context.Context, t *testing.T, e *httpexpect.Expect)
	// A function that returns the request headers (will be sent via "WithHeaders")
	PopulateHeaders func(ctx context.Context, t *testing.T) map[string]string
	// A function that returns the request query parameters (will be sent via "WithQuery")
	PopulateQueryParams func(ctx context.Context, t *testing.T) map[string]string
	// A function that returns the request cookies (will be sent via "WithCookies")
	PopulateCookies func(ctx context.Context, t *testing.T) map[string]string
	// Expected HTTP status code from the API call
	ExpHTTPStatus int
	// A function that returns a struct to send to the request via "WithJSON". Leave out if you don't need a JSON body
	PopulateJSON func(ctx context.Context, t *testing.T) interface{}
	// A function that validates the response. This should call t.Error as appropriate. Leave out if you don't need to validate.
	ValidateResponse func(ctx context.Context, t *testing.T, resp *httpexpect.Response) context.Context
	// A function that will be run immediately after the testcase validation
	AfterValidate func(ctx context.Context, t *testing.T, e *httpexpect.Expect)
}

func (apitest *APITest) setupTest(t *testing.T) func(t *testing.T) {
	t.Logf("Running Test Setup for %s", t.Name())

	if apitest.GetHTTPHandler != nil {
		apitest.server = httptest.NewServer(apitest.GetHTTPHandler(apitest.Ctx, t))
	}

	if apitest.SetupTest != nil {
		apitest.SetupTest(apitest.Ctx, t)
	}

	return func(t *testing.T) {
		if apitest.server != nil {
			defer apitest.server.Close()
		}
		t.Logf("Running Test Teardown for %s", t.Name())

		if apitest.TeardownTest != nil {
			apitest.TeardownTest(apitest.Ctx, t)
		}
	}
}

func (apitest *APITest) setupTestCase(t *testing.T, tc APITestCase) func(t *testing.T, tc APITestCase) {
	if apitest.SetupTestCase != nil {
		t.Logf("Running Test Case Setup for %s", tc.Name)
		apitest.SetupTestCase(apitest.Ctx, t, tc)
	}
	return func(t *testing.T, tc APITestCase) {
		if apitest.TeardownTestCase != nil {
			t.Logf("Running Test Case Teardown for %s", tc.Name)
			apitest.TeardownTestCase(apitest.Ctx, t, tc)
		}
	}
}

// RunTests is used to execute the testCases provided and call any
// setup/teardown steps as well as perform definted validation
func (apitest *APITest) RunTests(t *testing.T, testCases []APITestCase) {
	if apitest.GetHTTPHandler == nil && apitest.APIEndpointURL == "" {
		panic("One of GetHTTPHandler or APIEndpointURL must be specified")
	}

	teardownTest := apitest.setupTest(t)
	defer teardownTest(t)

	url := apitest.APIEndpointURL
	if apitest.server != nil {
		url = apitest.server.URL
	}

	for _, tt := range testCases {
		teardownTestCase := apitest.setupTestCase(t, tt)
		t.Run(tt.Name, func(t *testing.T) {
			e := httpexpect.New(t, url)

			if tt.BeforeRequest != nil {
				tt.BeforeRequest(apitest.Ctx, t, e)
			}

			req := e.Request(tt.ReqType, tt.ResourcePath(apitest.Ctx, t))

			if tt.PopulateHeaders != nil {
				req = req.WithHeaders(tt.PopulateHeaders(apitest.Ctx, t))
			}
			if tt.PopulateQueryParams != nil {
				for k, v := range tt.PopulateQueryParams(apitest.Ctx, t) {
					req = req.WithQuery(k, v)
				}
			}

			if tt.PopulateCookies != nil {
				req = req.WithCookies(tt.PopulateCookies(apitest.Ctx, t))
			}

			if tt.PopulateJSON != nil {
				req = req.WithJSON(tt.PopulateJSON(apitest.Ctx, t))
			}

			resp := req.Expect().Status(tt.ExpHTTPStatus)

			if tt.ValidateResponse != nil {
				apitest.Ctx = tt.ValidateResponse(apitest.Ctx, t, resp)
			}

			if tt.AfterValidate != nil {
				tt.AfterValidate(apitest.Ctx, t, e)
			}
		})
		teardownTestCase(t, tt)
	}
}
