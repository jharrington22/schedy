package migrate

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/example/resy-scheduler/internal/db"
)

//go:embed *.sql
var fs embed.FS

func Up(ctx context.Context, d *db.DB) error {
	entries, err := fs.ReadDir(".")
	if err != nil {
		return err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// schema_migrations table
	if err := d.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY);`); err != nil {
		return err
	}

	for _, f := range files {
		var applied bool
		if err := d.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, f).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}

		b, err := fs.ReadFile(f)
		if err != nil {
			return err
		}
		sql := string(b)

		if err := d.Exec(ctx, sql); err != nil {
			return fmt.Errorf("apply %s: %w", f, err)
		}
		if err := d.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, f); err != nil {
			return err
		}
	}

	return nil
}
