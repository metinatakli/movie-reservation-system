package integration_test

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	pgxstd "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresContainer struct {
	Container        *postgres.PostgresContainer
	ConnectionString string
}

type RedisContainer struct {
	Container        *tcredis.RedisContainer
	ConnectionString string
}

func getDbContainer(ctx context.Context) (*PostgresContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgis/postgis:17-3.4",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":                dbName,
			"POSTGRES_USER":              dbUser,
			"POSTGRES_PASSWORD":          dbPassword,
			"POSTGRES_INITDB_ARGS":       "--data-checksums",
			"POSTGRES_HOST_AUTH_METHOD":  "trust",
			"POSTGRES_INITDB_EXTRA_ARGS": "--create-extension=postgis",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
			wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
				return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
					dbUser, dbPassword, host, port.Port(), dbName)
			}),
		).WithDeadline(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start DB container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser,
		dbPassword,
		host,
		port.Port(),
		dbName,
	)

	err = runMigrations(connStr, "file://../../migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	dbContainer := &PostgresContainer{
		Container:        &postgres.PostgresContainer{Container: container},
		ConnectionString: connStr,
	}

	return dbContainer, nil
}

func runMigrations(dsn string, migrationsPath string) error {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	db := pgxstd.OpenDB(*config)
	defer db.Close()

	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		return fmt.Errorf("pgx migration driver error: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "pgx", driver)
	if err != nil {
		return fmt.Errorf("migrate.New error: %w", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

func getCacheContainer(ctx context.Context) (*RedisContainer, error) {
	container, err := tcredis.Run(ctx, cacheImageName)
	if err != nil {
		return nil, fmt.Errorf("failed to start cache container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	connStr := fmt.Sprintf("%s:%s", host, port.Port())

	cacheContainer := &RedisContainer{
		Container:        container,
		ConnectionString: connStr,
	}

	return cacheContainer, nil
}
