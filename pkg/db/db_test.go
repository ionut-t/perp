package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDDLQuery(t *testing.T) {
	t.Parallel()

	ddl := []struct {
		name  string
		query string
	}{
		// Tables
		{"create table", "CREATE TABLE users (id int)"},
		{"drop table", "DROP TABLE users"},
		{"alter table", "ALTER TABLE users ADD COLUMN name text"},
		// Indexes
		{"create index", "CREATE INDEX idx_users_email ON users (email)"},
		{"create unique index", "CREATE UNIQUE INDEX idx_users_email ON users (email)"},
		{"drop index", "DROP INDEX idx_users_email"},
		// Views
		{"create view", "CREATE VIEW active_users AS SELECT * FROM users"},
		{"create materialized view", "CREATE MATERIALIZED VIEW user_stats AS SELECT count(*) FROM users"},
		{"drop view", "DROP VIEW active_users"},
		{"drop materialized view", "DROP MATERIALIZED VIEW user_stats"},
		{"alter view", "ALTER VIEW active_users RENAME TO enabled_users"},
		// Schema
		{"create schema", "CREATE SCHEMA reporting"},
		{"drop schema", "DROP SCHEMA reporting"},
		// Types
		{"create type", "CREATE TYPE status AS ENUM ('active', 'inactive')"},
		{"drop type", "DROP TYPE status"},
		{"alter type", "ALTER TYPE status ADD VALUE 'pending'"},
		// Extensions
		{"create extension", "CREATE EXTENSION IF NOT EXISTS pgcrypto"},
		{"drop extension", "DROP EXTENSION pgcrypto"},
		// Truncate
		{"truncate", "TRUNCATE users"},
		// Case insensitivity
		{"uppercase", "CREATE TABLE upper (id int)"},
		{"mixed case", "Create Table mixed (id int)"},
	}

	notDDL := []struct {
		name  string
		query string
	}{
		{"select", "SELECT * FROM users"},
		{"insert", "INSERT INTO users (name) VALUES ('alice')"},
		{"update", "UPDATE users SET name = 'bob' WHERE id = 1"},
		{"delete", "DELETE FROM users WHERE id = 1"},
		{"begin", "BEGIN"},
		{"commit", "COMMIT"},
	}

	for _, tt := range ddl {
		t.Run("is DDL: "+tt.name, func(t *testing.T) {
			assert.True(t, isDDLQuery(tt.query), "expected %q to be detected as DDL", tt.query)
		})
	}

	for _, tt := range notDDL {
		t.Run("not DDL: "+tt.name, func(t *testing.T) {
			assert.False(t, isDDLQuery(tt.query), "expected %q to NOT be detected as DDL", tt.query)
		})
	}
}
