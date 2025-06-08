package integration_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	BaseSuite
}

func TestUserSuite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	suite.Run(t, new(UserTestSuite))
}

func (s *UserTestSuite) TestGetCurrentUser() {
	scenarios := []Scenario{
		{
			Name:           "returns 401 when user is not logged in",
			Method:         "GET",
			URL:            "/users/me",
			ExpectedStatus: 401,
			ExpectedResponse: `{
				"message": "You must be authenticated to access this resource"
			}`,
		},
		{
			Name:           "returns 404 when user ID in session but not found in DB",
			Method:         "GET",
			URL:            "/users/me",
			ExpectedStatus: 404,
			ExpectedResponse: `{
				"message": "The requested resource not found"
			}`,
			Cookies: s.app.authenticatedUserCookies(s.T()),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)
			},
		},
		{
			Name:           "successfully retrieves current user",
			Method:         "GET",
			URL:            "/users/me",
			ExpectedStatus: 200,
			ExpectedResponse: fmt.Sprintf(`{
				"id": 1,
				"firstName": "%s",
				"lastName": "%s",
				"email": "%s",
				"birthDate": "%s",
				"gender": "%s",
				"activated": true,
				"version": 1
			}`, TestUserFirstName, TestUserLastName, TestUserEmail, TestUserBirthDate, TestUserGender),
			Cookies: s.app.authenticatedUserCookies(s.T()),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				// Create a test user
				user := defaultTestUser()
				user.Activated = true
				insertTestUser(t, app.DB, user)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func (s *UserTestSuite) TestUpdateUser() {
	scenarios := []Scenario{
		{
			Name:           "returns 401 when user is not logged in",
			Method:         "PATCH",
			URL:            "/users/me",
			ExpectedStatus: 401,
			ExpectedResponse: `{
				"message": "You must be authenticated to access this resource"
			}`,
		},
		{
			Name:           "returns 400 for request with malformed JSON",
			Method:         "PATCH",
			URL:            "/users/me",
			Body:           strings.NewReader(`{"bad":"json"`),
			ExpectedStatus: 400,
			ExpectedResponse: `{
				"message": "body contains badly-formed JSON"
			}`,
			Cookies: s.app.authenticatedUserCookies(s.T()),
		},
		{
			Name:   "returns 422 for invalid input data",
			Method: "PATCH",
			URL:    "/users/me",
			Body: strings.NewReader(`{
				"firstName": "J",
				"lastName": "D",
				"birthDate": "2020-01-01",
				"gender": "INVALID"
			}`),
			ExpectedStatus: 422,
			ExpectedResponse: `{
				"message": "One or more fields have invalid values",
				"validationErrors": [
					{"field": "BirthDate", "issue": "must be at least 15 years old"},
					{"field": "FirstName", "issue": "must be at least 2 characters long"},
					{"field": "Gender", "issue": "is invalid"},
					{"field": "LastName", "issue": "must be at least 2 characters long"}
				]
			}`,
			Cookies: s.app.authenticatedUserCookies(s.T()),
		},
		{
			Name:           "returns 404 when user not found in DB",
			Method:         "PATCH",
			URL:            "/users/me",
			Body:           strings.NewReader(`{"firstName": "John"}`),
			ExpectedStatus: 404,
			ExpectedResponse: `{
				"message": "The requested resource not found"
			}`,
			Cookies: s.app.authenticatedUserCookies(s.T()),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)
			},
		},
		{
			Name:   "successfully updates user",
			Method: "PATCH",
			URL:    "/users/me",
			Body: strings.NewReader(fmt.Sprintf(`{
				"firstName": "John",
				"lastName": "Doe",
				"birthDate": "%s",
				"gender": "%s"
			}`, TestUserBirthDate, TestUserGender)),
			ExpectedStatus: 200,
			ExpectedResponse: fmt.Sprintf(`{
				"id": 1,
				"firstName": "John",
				"lastName": "Doe",
				"email": "%s",
				"birthDate": "%s",
				"gender": "%s",
				"activated": true,
				"version": 2
			}`, TestUserEmail, TestUserBirthDate, TestUserGender),
			Cookies: s.app.authenticatedUserCookies(s.T()),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				// Create a test user
				user := defaultTestUser()
				user.Activated = true
				insertTestUser(t, app.DB, user)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}
