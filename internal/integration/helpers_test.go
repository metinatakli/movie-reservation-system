package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"math/big"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/stretchr/testify/require"
)

var keysToIgnore = map[string]struct{}{
	"timestamp": {},
	"requestId": {},
	"createdAt": {},
}

func prepareRequest(method, path string, body io.Reader, headers map[string]string, cookies []http.Cookie) (*http.Request, error) {
	req := httptest.NewRequest(method, path, body)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	for _, cookie := range cookies {
		req.AddCookie(&cookie)
	}

	return req, nil
}

func compareResponse(t *testing.T, body io.Reader, expectedResponse string) {
	t.Helper()

	var actual map[string]any
	require.NoError(t, json.NewDecoder(body).Decode(&actual))

	cleanMap(actual)

	var expected map[string]any
	require.NoError(t, json.Unmarshal([]byte(expectedResponse), &expected))

	// ignore indetermistic fields while comparing
	opts := cmpopts.IgnoreMapEntries(func(k string, _ any) bool {
		return k == "timestamp" || k == "requestId" || k == "createdAt"
	})

	if diff := cmp.Diff(expected, actual, opts); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func cleanMap(m map[string]any) {
	for k := range m {
		if _, ok := keysToIgnore[k]; ok {
			delete(m, k)
			continue
		}
		if nested, ok := m[k].(map[string]any); ok {
			cleanMap(nested)
		}
	}
}

// defaultTestUser returns a *domain.User with default test values.
func defaultTestUser() *domain.User {
	var user domain.User

	user.FirstName = TestUserFirstName
	user.LastName = TestUserLastName
	user.Email = TestUserEmail
	user.BirthDate, _ = time.Parse("2006-01-02", TestUserBirthDate)
	user.Gender = TestUserGender
	user.Activated = false
	_ = user.Password.Set(TestUserPassword)

	return &user
}

// defaultTestToken returns a *domain.Token with default test values.
func defaultTestToken(userID int, scope string) *domain.Token {
	return &domain.Token{
		Plaintext: TestToken,
		Hash:      sha256Sum(TestToken),
		UserId:    int64(userID),
		Expiry:    time.Now().Add(24 * time.Hour),
		Scope:     scope,
	}
}

// insertTestUser inserts a user into the DB using the domain.User struct.
func insertTestUser(t testing.TB, db *pgxpool.Pool, user *domain.User) int {
	t.Helper()

	var userID int
	err := db.QueryRow(
		context.Background(),
		`INSERT INTO users (first_name, last_name, email, password_hash, birth_date, gender, activated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		user.FirstName,
		user.LastName,
		user.Email,
		user.Password.Hash,
		user.BirthDate,
		user.Gender,
		user.Activated,
	).Scan(&userID)

	require.NoError(t, err)

	return userID
}

// insertTestToken inserts a token into the DB using the domain.Token struct.
func insertTestToken(t testing.TB, db *pgxpool.Pool, token *domain.Token) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		`INSERT INTO tokens (hash, user_id, expiry, scope) VALUES ($1, $2, $3, $4)`,
		token.Hash,
		token.UserId,
		token.Expiry,
		token.Scope,
	)

	require.NoError(t, err)
}

// truncateUsersAndTokens truncates the users and tokens tables and resets their identity columns.
// It is used to clean up the database before each test to ensure a clean state.
func truncateUsersAndTokens(t testing.TB, db *pgxpool.Pool) {
	t.Helper()

	_, err := db.Exec(context.Background(), "TRUNCATE tokens RESTART IDENTITY CASCADE")
	require.NoError(t, err)
	_, err = db.Exec(context.Background(), "TRUNCATE users RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func sha256Sum(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

// authenticatedUserCookies creates a session cookie for an authenticated user with ID 1.
// This is used in tests to simulate an authenticated user session.
func (app *TestApp) authenticatedUserCookies(t *testing.T) []http.Cookie {
	ctx := context.Background()

	ctx, err := app.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	app.SessionManager.Put(ctx, "userID", TestUserId)

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

// insertTestMovie inserts a movie into the DB using the domain.Movie struct.
func insertTestMovie(t testing.TB, db *pgxpool.Pool, movie *domain.Movie) int {
	t.Helper()

	var movieID int
	err := db.QueryRow(
		context.Background(),
		`INSERT INTO movies (title, description, genres, language, release_date, duration, poster_url, director, cast_members, rating)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
		movie.Title,
		movie.Description,
		movie.Genres,
		movie.Language,
		movie.ReleaseDate,
		movie.Duration,
		movie.PosterUrl,
		movie.Director,
		movie.CastMembers,
		movie.Rating,
	).Scan(&movieID)

	require.NoError(t, err)

	return movieID
}

// truncateMovies truncates the movies table and resets its identity column.
func truncateMovies(t testing.TB, db *pgxpool.Pool) {
	t.Helper()

	_, err := db.Exec(context.Background(), "TRUNCATE movies RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

// defaultTestMovie returns a *domain.Movie with default test values.
func defaultTestMovie() *domain.Movie {
	releaseDate, _ := time.Parse("2006-01-02", TestMovieReleaseDate)
	rating := pgtype.Numeric{Int: new(big.Int).SetInt64(75), Exp: -1, Valid: true}
	return &domain.Movie{
		Title:       TestMovieTitle,
		Description: TestMovieDescription,
		Genres:      TestMovieGenres,
		Language:    TestMovieLanguage,
		ReleaseDate: releaseDate,
		Duration:    TestMovieDuration,
		PosterUrl:   TestMoviePosterUrl,
		Director:    TestMovieDirector,
		CastMembers: TestMovieCast,
		Rating:      rating,
	}
}

// executeSQLFile executes a SQL file at the given path against the provided DB.
func executeSQLFile(t testing.TB, db *pgxpool.Pool, filePath string) {
	t.Helper()

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)

	queries := strings.Split(string(content), ";")
	for _, query := range queries {
		q := strings.TrimSpace(query)
		if q == "" {
			continue
		}
		_, err := db.Exec(context.Background(), q)
		require.NoError(t, err)
	}
}
