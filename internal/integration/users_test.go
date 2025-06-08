package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/stretchr/testify/require"
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

func (s *UserTestSuite) TestInitiateUserDeletion() {
	scenarios := []Scenario{
		{
			Name:           "returns 401 when user is not logged in",
			Method:         "POST",
			URL:            "/users/me/deletion-request",
			ExpectedStatus: 401,
			ExpectedResponse: `{
				"message": "You must be authenticated to access this resource"
			}`,
		},
		{
			Name:           "returns 400 for request with malformed JSON",
			Method:         "POST",
			URL:            "/users/me/deletion-request",
			Body:           strings.NewReader(`{"bad":"json"`),
			ExpectedStatus: 400,
			ExpectedResponse: `{
				"message": "body contains badly-formed JSON"
			}`,
			Cookies: s.app.authenticatedUserCookies(s.T()),
		},
		{
			Name:           "returns 401 for invalid password",
			Method:         "POST",
			URL:            "/users/me/deletion-request",
			Body:           strings.NewReader(`{"password": "wrongpassword"}`),
			ExpectedStatus: 401,
			ExpectedResponse: `{
				"message": "Invalid email or password"
			}`,
			Cookies: s.app.authenticatedUserCookies(s.T()),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				// Create a test user
				user := defaultTestUser()
				user.Activated = true
				insertTestUser(t, app.DB, user)
			},
		},
		{
			Name:           "returns 404 when user not found in DB",
			Method:         "POST",
			URL:            "/users/me/deletion-request",
			Body:           strings.NewReader(fmt.Sprintf(`{"password": "%s"}`, TestUserPassword)),
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
			Name:             "successfully initiates user deletion",
			Method:           "POST",
			URL:              "/users/me/deletion-request",
			Body:             strings.NewReader(fmt.Sprintf(`{"password": "%s"}`, TestUserPassword)),
			ExpectedStatus:   202,
			ExpectedResponse: ``,
			Cookies:          s.app.authenticatedUserCookies(s.T()),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				// Create a test user
				user := defaultTestUser()
				user.Activated = true
				insertTestUser(t, app.DB, user)

				app.Mailer.Reset()
			},
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				// Verify that deletion token has been created
				var tokenCount int
				err := app.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM tokens WHERE user_id = $1 AND scope = $2", 1, domain.UserDeletionScope).Scan(&tokenCount)
				require.NoError(t, err)
				require.Equal(t, 1, tokenCount)

				// Verify that email was sent
				emails := app.Mailer.GetSentEmails()
				require.Len(t, emails, 1)

				email := emails[0]
				require.Equal(t, TestUserEmail, email.Recipient)
				require.Equal(t, "user_deletion.tmpl", email.TemplateFile)

				data, ok := email.Data.(map[string]any)
				require.True(t, ok)
				require.Equal(t, 1, data["userID"])
				require.NotEmpty(t, data["deletionToken"])
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}
