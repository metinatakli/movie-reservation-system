package integration_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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
			Body: strings.NewReader(`{
				"email": "test@example.com",
				"firstName": "John",
				"lastName": "Doe",
				"password": "Test123!@#",
				"birthDate": "1990-01-01",
				"gender": "M"
			}`),
			ExpectedStatus: 400,
			ExpectedResponse: `{
				"message": "invalid input data"
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateUsersAndTokens(t, app.DB)

				_, err := app.DB.Exec(context.Background(), `
					INSERT INTO users (first_name, last_name, email, password_hash, birth_date, gender, activated)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
				`, "Existing", "User", "test@example.com", "$2a$12$1qAz2wSx3eDc4rFv5tGb5e", "1990-01-01", "M", false)
				require.NoError(t, err)

				app.Mailer.Reset()
			},
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				// Verify that no new user was created
				var userCount int
				err := app.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE email = $1", "test@example.com").Scan(&userCount)
				require.NoError(t, err)
				require.Equal(t, 1, userCount, "should not create a new user")

				// Verify that no new activation token was created
				var tokenCount int
				err = app.DB.QueryRow(
					context.Background(),
					"SELECT COUNT(*) FROM tokens WHERE user_id IN (SELECT id FROM users WHERE email = $1) AND scope = $2", "test@example.com", "user_activation").Scan(&tokenCount)
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
			Body: strings.NewReader(`{
				"email": "test@example.com",
				"firstName": "John",
				"lastName": "Doe",
				"password": "Test123!@#",
				"birthDate": "1990-01-01",
				"gender": "M"
			}`),
			ExpectedStatus: 202,
			ExpectedResponse: `{
				"id": 1,
				"firstName": "John",
				"lastName": "Doe",
				"email": "test@example.com",
				"birthDate": "1990-01-01",
				"gender": "M",
				"activated": false,
				"version": 1
			}`,
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
				err := app.DB.QueryRow(context.Background(), "SELECT id, email, activated FROM users WHERE email = $1", "test@example.com").Scan(
					&user.ID, &user.Email, &user.Activated,
				)
				require.NoError(t, err)
				require.Equal(t, "test@example.com", user.Email)
				require.False(t, user.Activated)

				// Verify that activation has been created
				var tokenCount int
				err = app.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM tokens WHERE user_id = $1 AND scope = $2", user.ID, "user_activation").Scan(&tokenCount)
				require.NoError(t, err)
				require.Equal(t, 1, tokenCount)

				// Verify that email sent part was triggered
				emails := app.Mailer.GetSentEmails()
				require.Len(t, emails, 1)

				email := emails[0]
				require.Equal(t, "test@example.com", email.Recipient)
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

func truncateUsersAndTokens(t testing.TB, db *pgxpool.Pool) {
	_, err := db.Exec(context.Background(), "TRUNCATE tokens RESTART IDENTITY CASCADE")
	require.NoError(t, err)
	_, err = db.Exec(context.Background(), "TRUNCATE users RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}
