package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/yourusername/2gis-parser/internal/models"
)

type DB struct {
	db *sql.DB
}

func New(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	store := &DB{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("database migration: %w", err)
	}

	log.Println("✅ Database ready:", dbPath)
	return store, nil
}

func (s *DB) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS companies (
			id            TEXT PRIMARY KEY,
			name          TEXT NOT NULL,
			address       TEXT,
			city          TEXT,
			phone         TEXT,
			website       TEXT,
			category      TEXT,
			working_hours TEXT,
			lat           REAL,
			lon           REAL,
			query         TEXT,
			parsed_at     DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS parse_sessions (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL,
			query      TEXT,
			cities     TEXT,
			total      INTEGER DEFAULT 0,
			status     TEXT DEFAULT 'running',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

// SaveCompanies saves or updates a list of companies.
func (s *DB) SaveCompanies(companies []models.Company, city, query string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO companies
			(id, name, address, city, phone, website, category, lat, lon, query)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range companies {
		_, err := stmt.Exec(c.ID, c.Name, c.Address, city, c.Phone, c.Website, c.Category, c.Lat, c.Lon, query)
		if err != nil {
			log.Printf("Failed to save %s: %v", c.Name, err)
		}
	}

	return tx.Commit()
}

// GetByQuery returns companies from the database by query and city.
func (s *DB) GetByQuery(query, city string) ([]models.Company, error) {
	rows, err := s.db.Query(`
		SELECT id, name, address, city, phone, website, category, lat, lon
		FROM companies
		WHERE query = ? AND city = ?
		ORDER BY name
	`, query, city)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Company
	for rows.Next() {
		var c models.Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Address, &c.City, &c.Phone, &c.Website, &c.Category, &c.Lat, &c.Lon); err != nil {
			continue
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *DB) Close() {
	s.db.Close()
}
