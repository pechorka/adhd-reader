package bootstrap

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/pechorka/stdlib/pkg/errs"

	_ "github.com/mattn/go-sqlite3"
	"go.etcd.io/bbolt"
)

type BboltConfig struct {
	Path string `envconfig:"PATH" required:"true"`
}

func Bbolt(cfg BboltConfig) (*bbolt.DB, error) {
	db, err := bbolt.Open(cfg.Path, os.ModePerm, nil)
	if err != nil {
		return nil, errs.Wrap(err, "failed to open bolt db")
	}

	return db, nil
}

type SqliteConfig struct {
	Path     string `envconfig:"PATH" required:"true"`
	InMemory bool   `envconfig:"IN_MEMORY" default:false`
}

func (cfg SqliteConfig) Mode() string {
	if cfg.InMemory {
		return "memory"
	}

	return "rwc"
}

func Sqlite(cfg SqliteConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(`file:%s?_fk=1&mode%s`,
		cfg.Path,
		cfg.Mode(),
	)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, errs.Wrap(err, "failed")
	}

	return db, nil
}

func applyMigrations(db *sql.DB) error {
	migration := `
	CREATE TABLE IF NOT EXISTS user_meta (
		id TEXT PRIMARY KEY,
		created_at INTEGER NOT NULL,
		name TEXT NOT NULL,
		language TEXT NOT NULL,
		gamification_user_id TEXT
	);
	
	CREATE TABLE IF NOT EXISTS user_preferences (
		user_id TEXT NOT NULL,
		chunk_size INTEGER NOT NULL,
		gamification_enabled INTEGER, 
		report_every TEXT,
		updated_at INTEGER,
		FOREIGN KEY(user_id) REFERENCES user_meta(id)
	);
	
	CREATE TABLE IF NOT EXISTS text (
		id TEXT PRIMARY KEY,
		name TEXT,
		source TEXT,
		current_chunk INTEGER,
		created_at TEXT,
		modified_at TEXT,
		deleted INTEGER,
		meta_for_analytics TE
	);
	
	CREATE TABLE IF NOT EXISTS processed_files (
		checksum TEXT PRIMARY KEY,
		bucket_name TEXT,
		chunk_size INT,
		user_id UUID,
		FOREIGN KEY(user_id) REFERENCES user_meta(id)
	);
	
	CREATE TABLE IF NOT EXISTS processed_urls (
		url TEXT PRIMARY KEY,
		bucket_name TEXT,
		chunk_size INT,
		user_id UUID,
		FOREIGN KEY(user_id) REFERENCES user_meta(id)
	);
	`

	_, err := db.Exec(migration)
	return err
}
