package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const (
	createTemplatesTable = `
	CREATE TABLE IF NOT EXISTS templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		template_id TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		severity TEXT,
		tags TEXT,
		author TEXT,
		file_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_templates_severity ON templates(severity);
	CREATE INDEX IF NOT EXISTS idx_templates_tags ON templates(tags);
	`

	createScanTasksTable = `
	CREATE TABLE IF NOT EXISTS scan_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		status TEXT NOT NULL,
		pocs TEXT NOT NULL,
		targets TEXT NOT NULL,
		total_requests INTEGER DEFAULT 0,
		completed_requests INTEGER DEFAULT 0,
		found_vulns INTEGER DEFAULT 0,
		output_file TEXT,
		start_time DATETIME,
		end_time DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_scan_tasks_status ON scan_tasks(status);
	`
)

type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dbPath string) (*Database, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize database
	database := &Database{db: db}
	if err := database.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return database, nil
}

// initTables creates database tables if they don't exist
func (d *Database) initTables() error {
	// Create templates table
	if _, err := d.db.Exec(createTemplatesTable); err != nil {
		return fmt.Errorf("failed to create templates table: %w", err)
	}

	// Create scan_tasks table
	if _, err := d.db.Exec(createScanTasksTable); err != nil {
		return fmt.Errorf("failed to create scan_tasks table: %w", err)
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// GetDB returns the underlying database connection
func (d *Database) GetDB() *sql.DB {
	return d.db
}
