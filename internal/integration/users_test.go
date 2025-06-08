package integration_test

import (
	"fmt"
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
