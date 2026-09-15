package main

import (
	"database/sql"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

const dbPath = "bca_finance.sqlite"

func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	DB = db
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	log.Printf("sqlite connected: %s", dbPath)
	return db, nil
}

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		nik TEXT NOT NULL UNIQUE,
		phone TEXT,
		address TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS vehicles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dealer TEXT NOT NULL,
		brand TEXT NOT NULL,
		model TEXT NOT NULL,
		price REAL NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS applications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		customer_id INTEGER NOT NULL REFERENCES customers(id),
		vehicle_id INTEGER NOT NULL REFERENCES vehicles(id),
		created_by INTEGER NOT NULL REFERENCES users(id),
		down_payment REAL NOT NULL,
		tenor INTEGER NOT NULL,
		installment REAL NOT NULL,
		status TEXT NOT NULL DEFAULT 'DRAFT',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS application_documents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		application_id INTEGER NOT NULL REFERENCES applications(id),
		document_type TEXT NOT NULL,
		file_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS application_approvals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		application_id INTEGER NOT NULL REFERENCES applications(id),
		approver_id INTEGER NOT NULL REFERENCES users(id),
		status TEXT NOT NULL,
		notes TEXT,
		approved_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	return seedInitialData(db)
}

func seedInitialData(db *sql.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`
		INSERT INTO users (username, password_hash, role)
		VALUES (?, ?, 'SALES')
		ON CONFLICT (username) DO NOTHING
	`, "sales1", string(hash))

	_, _ = db.Exec(`
		INSERT INTO users (username, password_hash, role)
		VALUES (?, ?, 'APPROVER')
		ON CONFLICT (username) DO NOTHING
	`, "approver1", string(hash))

	_, _ = db.Exec(`
		INSERT INTO vehicles (id, dealer, brand, model, price) VALUES
		(1, 'Dealer Toyota BSD', 'Toyota', 'Avanza 1.5 G CVT', 275000000),
		(2, 'Dealer Honda Bintaro', 'Honda', 'Brio Satya E CVT', 198000000),
		(3, 'Dealer Mitsubishi Serpong', 'Mitsubishi', 'Xpander Ultimate CVT', 315000000)
		ON CONFLICT (id) DO NOTHING
	`)

	return nil
}
