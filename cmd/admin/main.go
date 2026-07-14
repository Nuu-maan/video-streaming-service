// Command admin performs the operator tasks that previously required raw SQL:
// creating accounts (including the very first admin) and changing a user's
// role. It loads internal/config, so it reads the exact same DB_* environment
// as the API and worker and cannot quietly point at a different database.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository/postgres"
	"github.com/Nuu-maan/video-streaming-service/pkg/security"
)

// version is stamped by the linker in the Docker build.
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("a subcommand is required")
	}

	switch args[0] {
	case "promote":
		return promote(args[1:])
	case "create":
		return create(args[1:])
	case "version":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: admin <subcommand> [flags]

Subcommands:
  promote --username <name> --role <role>
      Change an existing user's role.

  create --username <name> --email <addr> --password <pw> [--role <role>]
      Create a user. The password may instead be supplied via the
      ADMIN_PASSWORD environment variable to keep it out of shell history.

  version
      Print the build version.

Roles: guest, user, premium, moderator, admin

Connection settings come from the same DB_* environment (and .env file) the
API server reads.
`)
}

func promote(args []string) error {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	username := fs.String("username", "", "username of the account to change")
	role := fs.String("role", "", "target role")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *username == "" || *role == "" {
		return errors.New("promote requires --username and --role")
	}

	newRole := domain.Role(*role)
	if !newRole.IsValid() {
		return fmt.Errorf("%w: %q", domain.ErrInvalidRole, *role)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	users, pool, err := connect(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	user, err := users.GetByUsername(ctx, *username)
	if err != nil {
		return fmt.Errorf("looking up %q: %w", *username, err)
	}
	if user.Role == newRole {
		fmt.Printf("%s already has role %s\n", user.Username, newRole)
		return nil
	}

	previous := user.Role
	user.Role = newRole
	if err := users.Update(ctx, user); err != nil {
		return fmt.Errorf("updating role for %q: %w", *username, err)
	}

	fmt.Printf("%s: %s -> %s\n", user.Username, previous, newRole)
	return nil
}

func create(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	username := fs.String("username", "", "username for the new account")
	email := fs.String("email", "", "email address for the new account")
	password := fs.String("password", "", "password (or set ADMIN_PASSWORD)")
	role := fs.String("role", string(domain.RoleUser), "role for the new account")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *password == "" {
		*password = os.Getenv("ADMIN_PASSWORD")
	}
	if *username == "" || *email == "" || *password == "" {
		return errors.New("create requires --username, --email and --password (or ADMIN_PASSWORD)")
	}

	newRole := domain.Role(*role)
	if !newRole.IsValid() {
		return fmt.Errorf("%w: %q", domain.ErrInvalidRole, *role)
	}

	// HashPassword enforces the same strength policy as registration, so an
	// account minted at the terminal is held to the same standard.
	hash, err := security.HashPassword(*password)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user, err := domain.NewUser(*username, *email, hash, newRole)
	if err != nil {
		return fmt.Errorf("validating new user: %w", err)
	}
	// There is no verification email to click at the terminal; the operator
	// running this command vouches for the address. Without this the account
	// could log in but never upload (CanUploadVideos requires a verified email).
	user.EmailVerified = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	users, pool, err := connect(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := users.Create(ctx, user); err != nil {
		return fmt.Errorf("creating user %q: %w", *username, err)
	}

	fmt.Printf("created %s <%s> with role %s (id %s)\n", user.Username, user.Email, user.Role, user.ID)
	return nil
}

// connect loads the shared configuration and opens a pool against the same
// database the server uses. The caller owns closing the returned pool.
func connect(ctx context.Context) (*postgres.UserRepository, *pgxpool.Pool, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}

	pool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		return nil, nil, fmt.Errorf("creating connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("pinging database at %s: %w", cfg.Database.Host, err)
	}

	return postgres.NewUserRepository(pool), pool, nil
}
