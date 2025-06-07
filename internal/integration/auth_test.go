package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AuthTestSuite struct {
	BaseSuite
}

func TestAuthSuite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	suite.Run(t, new(AuthTestSuite))
}

func (s *AuthTestSuite) TestRegisterUser() {
	scenarios := []Scenario{
		{
			Name:             "returns 400 for request with malformed JSON",
			Method:           "POST",
			URL:              "/users",
			Body:             strings.NewReader(`{"bad":"json"`),
			ExpectedStatus:   400,
			ExpectedResponse: `{"message": "body contains badly-formed JSON"}`,
		},
		{
			Name:   "returns 422 for invalid input data",
			Method: "POST",
			URL:    "/users",
			Body: strings.NewReader(`{
				"email": "invalid-email",
				"firstName": "J",
				"lastName": "D",
				"password": "123",
				"birthDate": "2020-01-01",
				"gender": "INVALID"
			}`),
			ExpectedStatus: 422,
			ExpectedResponse: `{
				"message": "One or more fields have invalid values",
				"validationErrors": [
					{"field": "BirthDate", "issue": "must be at least 15 years old"},
					{"field": "Email", "issue": "must be a valid email address"},
					{"field": "FirstName", "issue": "must be at least 2 characters long"},
					{"field": "Gender", "issue": "is invalid"},
					{"field": "LastName", "issue": "must be at least 2 characters long"},
					{"field": "Password", "issue": "must be at least 8 characters long and include at least one uppercase letter, one lowercase letter, one number, and one special character (!@#$%^&*)."}
				]
			}`,
		},
		{
			Name:   "returns 400 when email already exists",
			Method: "POST",
			URL:    "/users",
			Body: strings.NewReader(fmt.Sprintf(`{
				"email": "%s",
				"firstName": "%s",
				"lastName": "%s",
				"password": "%s",
				"birthDate": "%s",
				"gender": "%s"
			}`, TestUserEmail, TestUserFirstName, TestUserLastName, TestUserPassword, TestUserBirthDate, TestUserGender)),
			ExpectedStatus: 400,
			ExpectedResponse: `{
				"message": "invalid input data"
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				// Create an existing user
				user := defaultTestUser()
				user.Activated = false
				insertTestUser(t, app.DB, user)

				app.Mailer.Reset()
			},
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				// Verify that no new user was created
				var userCount int
				err := app.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE email = $1", TestUserEmail).Scan(&userCount)
				require.NoError(t, err)
				require.Equal(t, 1, userCount, "should not create a new user")

				// Verify that no new activation token was created
				var tokenCount int
				err = app.DB.QueryRow(
					context.Background(),
					"SELECT COUNT(*) FROM tokens WHERE user_id IN (SELECT id FROM users WHERE email = $1) AND scope = $2",
					TestUserEmail,
					TestTokenScope,
				).Scan(&tokenCount)
				require.NoError(t, err)
				require.Equal(t, 0, tokenCount, "should not create a new activation token")

				// Verify that no email was not triggered
				emails := app.Mailer.GetSentEmails()
				require.Empty(t, emails, "should not send any emails")
			},
		},
		{
			Name:   "successfully registers a new user",
			Method: "POST",
			URL:    "/users",
			Body: strings.NewReader(fmt.Sprintf(`{
				"email": "%s",
				"firstName": "%s",
				"lastName": "%s",
				"password": "%s",
				"birthDate": "%s",
				"gender": "%s"
			}`, TestUserEmail, TestUserFirstName, TestUserLastName, TestUserPassword, TestUserBirthDate, TestUserGender)),
			ExpectedStatus: 202,
			ExpectedResponse: fmt.Sprintf(`{
				"id": 1,
				"firstName": "%s",
				"lastName": "%s",
				"email": "%s",
				"birthDate": "%s",
				"gender": "%s",
				"activated": false,
				"version": 1
			}`, TestUserFirstName, TestUserLastName, TestUserEmail, TestUserBirthDate, TestUserGender),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				app.Mailer.Reset()
			},
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				// Verify that new user has been created
				var user struct {
					ID        int
					Email     string
					Activated bool
				}
				err := app.DB.QueryRow(context.Background(), "SELECT id, email, activated FROM users WHERE email = $1", TestUserEmail).Scan(
					&user.ID, &user.Email, &user.Activated,
				)
				require.NoError(t, err)
				require.Equal(t, TestUserEmail, user.Email)
				require.False(t, user.Activated)

				// Verify that activation has been created
				var tokenCount int
				err = app.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM tokens WHERE user_id = $1 AND scope = $2", user.ID, TestTokenScope).Scan(&tokenCount)
				require.NoError(t, err)
				require.Equal(t, 1, tokenCount)

				// Verify that email sent part was triggered
				emails := app.Mailer.GetSentEmails()
				require.Len(t, emails, 1)

				email := emails[0]
				require.Equal(t, TestUserEmail, email.Recipient)
				require.Equal(t, "user_welcome.tmpl", email.TemplateFile)

				data, ok := email.Data.(map[string]any)
				require.True(t, ok)
				require.Equal(t, user.ID, data["userID"])
				require.NotEmpty(t, data["activationToken"])
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func (s *AuthTestSuite) TestActivateUser() {
	scenarios := []Scenario{
		{
			Name:             "returns 400 for request with malformed JSON",
			Method:           "PUT",
			URL:              "/users/activation",
			Body:             strings.NewReader(`{"bad":"json"`),
			ExpectedStatus:   400,
			ExpectedResponse: `{"message": "body contains badly-formed JSON"}`,
		},
		{
			Name:   "returns 422 for invalid input data",
			Method: "PUT",
			URL:    "/users/activation",
			Body: strings.NewReader(`{
				"token": "invalid-token"
			}`),
			ExpectedStatus: 422,
			ExpectedResponse: `{
				"message": "One or more fields have invalid values",
				"validationErrors": [
					{"field": "Token", "issue": "is invalid"}
				]
			}`,
		},
		{
			Name:   "returns 404 for non-existent token",
			Method: "PUT",
			URL:    "/users/activation",
			Body: strings.NewReader(fmt.Sprintf(`{
				"token": "%s"
			}`, TestActivationToken)),
			ExpectedStatus: 404,
			ExpectedResponse: `{
				"message": "The requested resource not found"
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)
			},
		},
		{
			Name:   "returns 409 for already activated user",
			Method: "PUT",
			URL:    "/users/activation",
			Body: strings.NewReader(fmt.Sprintf(`{
				"token": "%s"
			}`, TestActivationToken)),
			ExpectedStatus: 409,
			ExpectedResponse: `{
				"message": "Unable to update the record due to an edit conflict, please try again"
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				// Create an activated user
				user := defaultTestUser()
				user.Activated = true
				userID := insertTestUser(t, app.DB, user)

				// Create activation token for the user
				token := defaultTestToken(userID)
				insertTestToken(t, app.DB, token)
			},
		},
		{
			Name:   "successfully activates a user",
			Method: "PUT",
			URL:    "/users/activation",
			Body: strings.NewReader(fmt.Sprintf(`{
				"token": "%s"
			}`, TestActivationToken)),
			ExpectedStatus: 200,
			ExpectedResponse: `{
				"activated": true
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				// Create an unactivated user
				user := defaultTestUser()
				userID := insertTestUser(t, app.DB, user)

				// Create activation token for the user
				token := defaultTestToken(userID)
				insertTestToken(t, app.DB, token)
			},
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				// Verify that user is now activated
				var activated bool
				err := app.DB.QueryRow(context.Background(), "SELECT activated FROM users WHERE id = $1", 1).Scan(&activated)
				require.NoError(t, err)
				require.True(t, activated, "user should be activated")

				// Verify that activation token is deleted
				var tokenCount int
				err = app.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM tokens WHERE user_id = $1 AND scope = $2", 1, TestTokenScope).Scan(&tokenCount)
				require.NoError(t, err)
				require.Equal(t, 0, tokenCount, "activation token should be deleted")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func (s *AuthTestSuite) TestLogin() {
	scenarios := []Scenario{
		{
			Name:             "returns 400 for request with malformed JSON",
			Method:           "POST",
			URL:              "/sessions",
			Body:             strings.NewReader(`{"bad":"json"`),
			ExpectedStatus:   400,
			ExpectedResponse: `{"message": "body contains badly-formed JSON"}`,
		},
		{
			Name:   "returns 401 for invalid input data",
			Method: "POST",
			URL:    "/sessions",
			Body: strings.NewReader(`{
				"email": "invalid-email",
				"password": "123"
			}`),
			ExpectedStatus: 401,
			ExpectedResponse: `{
				"message": "Invalid email or password"
			}`,
		},
		{
			Name:   "returns 401 for non-existent user",
			Method: "POST",
			URL:    "/sessions",
			Body: strings.NewReader(`{
				"email": "nonexistent@example.com",
				"password": "Test123!@#"
			}`),
			ExpectedStatus: 401,
			ExpectedResponse: `{
				"message": "Invalid email or password"
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)
			},
		},
		{
			Name:   "returns 401 for incorrect password",
			Method: "POST",
			URL:    "/sessions",
			Body: strings.NewReader(fmt.Sprintf(`{
				"email": "%s",
				"password": "WrongPass123!@#"
			}`, TestUserEmail)),
			ExpectedStatus: 401,
			ExpectedResponse: `{
				"message": "Invalid email or password"
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				// Create a user with a known password
				user := defaultTestUser()
				insertTestUser(t, app.DB, user)
			},
		},
		{
			Name:   "returns 200 when user is already logged in",
			Method: "POST",
			URL:    "/sessions",
			Body: strings.NewReader(fmt.Sprintf(`{
				"email": "%s",
				"password": "%s"
			}`, TestUserEmail, TestUserPassword)),
			ExpectedStatus: 200,
			ExpectedResponse: `{
				"message": "You are already logged in"
			}`,
			Cookies: s.app.authenticatedUserCookies(s.T()),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)
			},
		},
		{
			// TODO: migrate cart data logic should be tested
			Name:   "successfully logs in a user",
			Method: "POST",
			URL:    "/sessions",
			Body: strings.NewReader(fmt.Sprintf(`{
				"email": "%s",
				"password": "%s"
			}`, TestUserEmail, TestUserPassword)),
			ExpectedStatus: 204,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				user := defaultTestUser()
				user.Activated = true
				insertTestUser(t, app.DB, user)
			},
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				// Verify that session cookie is set in response
				var sessionCookie *http.Cookie
				for _, cookie := range res.Cookies() {
					if cookie.Name == app.SessionManager.Cookie.Name {
						sessionCookie = cookie
						break
					}
				}
				require.NotNil(t, sessionCookie, "session cookie must be set")

				// Verify that user's info is loaded session
				ctx, err := app.SessionManager.Load(context.Background(), sessionCookie.Value)
				require.NoError(t, err)

				userID := app.SessionManager.GetInt(ctx, "userID")
				require.Equal(t, 1, userID, "user should be logged in")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func (s *AuthTestSuite) TestLogout() {
	scenarios := []Scenario{
		{
			Name:           "returns 404 when user is not logged in",
			Method:         "DELETE",
			URL:            "/sessions",
			ExpectedStatus: 404,
			ExpectedResponse: `{
				"message": "The requested resource not found"
			}`,
		},
		{
			Name:           "returns 204 when user is successfully logged out",
			Method:         "DELETE",
			URL:            "/sessions",
			ExpectedStatus: 204,
			Cookies:        s.app.authenticatedUserCookies(s.T()),
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				// Verify that session cookie is set in response
				var sessionCookie *http.Cookie
				for _, cookie := range res.Cookies() {
					if cookie.Name == app.SessionManager.Cookie.Name {
						sessionCookie = cookie
						break
					}
				}
				require.NotNil(t, sessionCookie, "session cookie must be set")
				require.True(t, sessionCookie.Expires.Before(time.Now()), "response cookie should have an expiry in the past")

				// Verify that user's info is removed from Redis
				redisKey := fmt.Sprintf("scs:session:%s", sessionCookie.Value)

				err := app.RedisClient.Get(context.Background(), redisKey).Err()
				require.ErrorIs(t, err, redis.Nil, "session key must be deleted from Redis after logout")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func (app *TestApp) authenticatedUserCookies(t *testing.T) []http.Cookie {
	ctx := context.Background()

	ctx, err := app.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	app.SessionManager.Put(ctx, "userID", 1)

	token, expiry, err := app.SessionManager.Commit(ctx)
	require.NoError(t, err)

	return []http.Cookie{
		{
			Name:    app.SessionManager.Cookie.Name,
			Value:   token,
			Expires: expiry,
			Path:    "/",
		},
	}
}
