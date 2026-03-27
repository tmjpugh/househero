package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func New(host, port, user, password, dbname string) (*DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	sqlDB, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	err = sqlDB.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("Database connected successfully")
	return &DB{sqlDB}, nil
}

func (db *DB) RunMigrations() error {
	migrationSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS homes (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		address TEXT NOT NULL,
		photo TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS inventory_items (
		id SERIAL PRIMARY KEY,
		home_id INTEGER NOT NULL REFERENCES homes(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(100) NOT NULL,
		make VARCHAR(100) NOT NULL,
		model VARCHAR(100),
		room VARCHAR(100) NOT NULL,
		serial_number VARCHAR(255),
		purchase_date DATE,
		warranty_expires DATE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tickets (
		id SERIAL PRIMARY KEY,
		home_id INTEGER NOT NULL REFERENCES homes(id) ON DELETE CASCADE,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		type VARCHAR(100) NOT NULL,
		priority VARCHAR(20) NOT NULL,
		status VARCHAR(20) NOT NULL,
		requester VARCHAR(255) NOT NULL,
		room VARCHAR(100) NOT NULL,
		inventory_item_id INTEGER REFERENCES inventory_items(id) ON DELETE SET NULL,
		inventory_item VARCHAR(255),
		estimated_cost VARCHAR(50),
		closer VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		closed_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ticket_dependencies (
		id SERIAL PRIMARY KEY,
		ticket_id INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
		blocked_by_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE,
		is_blocking_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS comments (
		id SERIAL PRIMARY KEY,
		ticket_id INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
		text TEXT NOT NULL,
		author VARCHAR(255) NOT NULL,
		is_system BOOLEAN DEFAULT FALSE,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS photos (
		id SERIAL PRIMARY KEY,
		ticket_id INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
		url TEXT NOT NULL,
		name VARCHAR(255) NOT NULL,
		uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS documents (
		id SERIAL PRIMARY KEY,
		inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
		doc_type VARCHAR(50) NOT NULL,
		name VARCHAR(255) NOT NULL,
		url TEXT NOT NULL,
		uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS notes (
		id SERIAL PRIMARY KEY,
		inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
		text TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_homes_user_id ON homes(user_id);
	CREATE INDEX IF NOT EXISTS idx_tickets_home_id ON tickets(home_id);
	CREATE INDEX IF NOT EXISTS idx_inventory_home_id ON inventory_items(home_id);
	`

	_, err := db.Exec(migrationSQL)
	return err
}
