// Sentinel migration runner.
//
// Subcommands:
//
//	migrate up               apply all pending migrations
//	migrate down             roll back the most recent migration
//	migrate status           print current schema_migrations version (and dirty flag)
//	migrate version          same as status
//	migrate force <N>        set version to N (use to recover from a dirty state)
//
// MySQL DSN is built from `config.Load()` (the same source the server uses), so
// `make migrate` Just Works as long as `.env` is populated. Override via
// MIGRATE_DATABASE_URL or MIGRATE_SOURCE_URL env vars if needed.
//
// Migrations live as `NNNN_*.sql` files under `backend/migrations/`. The schema
// version is tracked in the `schema_migrations` table — auto-created on first up.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql" // mysql driver
	_ "github.com/golang-migrate/migrate/v4/source/file"    // file source

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	srcURL, err := resolveSourceURL()
	if err != nil {
		fail("%v", err)
	}

	dbURL, err := resolveDBURL()
	if err != nil {
		fail("%v", err)
	}

	m, err := migrate.New(srcURL, dbURL)
	if err != nil {
		fail("migrate new: %v", err)
	}
	defer func() {
		serr, derr := m.Close()
		if serr != nil {
			fmt.Fprintf(os.Stderr, "source close: %v\n", serr)
		}
		if derr != nil {
			fmt.Fprintf(os.Stderr, "db close: %v\n", derr)
		}
	}()

	switch os.Args[1] {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fail("up: %v", err)
		}
		fmt.Println("up: applied")
	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fail("down: %v", err)
		}
		fmt.Println("down: applied")
	case "status", "version":
		v, dirty, err := m.Version()
		switch {
		case errors.Is(err, migrate.ErrNilVersion):
			fmt.Println("version: none")
		case err != nil:
			fail("version: %v", err)
		default:
			fmt.Printf("version: %d (dirty=%v)\n", v, dirty)
		}
	case "force":
		if len(os.Args) < 3 {
			fmt.Println("usage: migrate force <N>")
			os.Exit(2)
		}
		n, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fail("force: bad N %q: %v", os.Args[2], err)
		}
		if err := m.Force(n); err != nil {
			fail("force: %v", err)
		}
		fmt.Printf("force: version=%d\n", n)
	default:
		usage()
		os.Exit(2)
	}
}

// resolveSourceURL returns the golang-migrate `file://` URL for the migrations
// directory. Two-slash form (`file://<rel>`) is parsed as host=rel, path=""
// which makes the source silently ReadDir(".") and fail with
// "first .: file does not exist". We always emit the three-slash form so the
// URL parses as host="", path=<abs-path>.
func resolveSourceURL() (string, error) {
	if v := os.Getenv("MIGRATE_SOURCE_URL"); v != "" {
		// Normalise user-supplied URL to the three-slash form when the path
		// is a local filesystem path (no host).
		if strings.HasPrefix(v, "file://") && !strings.HasPrefix(v, "file:///") {
			v = "file:///" + strings.TrimPrefix(v, "file://")
		}
		return v, nil
	}
	migDir, err := findMigrationsDir()
	if err != nil {
		return "", err
	}
	return "file:///" + filepath.ToSlash(migDir), nil
}

func resolveDBURL() (string, error) {
	if v := os.Getenv("MIGRATE_DATABASE_URL"); v != "" {
		return v, nil
	}
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	// multiStatements=true is required because a single .sql file may contain
	// multiple `CREATE TABLE` statements; the driver defaults to one-statement
	// and would silently truncate on the first `;`.
	return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?multiStatements=true",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database), nil
}

// findMigrationsDir walks up from the current working directory until it finds
// a `migrations` directory. This makes `go run ./cmd/migrate up` from any
// subdirectory behave the same as running it from `backend/`.
func findMigrationsDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no `migrations` directory found from %s upward", cwd)
		}
		dir = parent
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrate <up|down|status|version|force <N>>")
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "migrate: "+format+"\n", args...)
	os.Exit(1)
}
