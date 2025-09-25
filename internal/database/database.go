package database

import (
	"database/sql"
	"log"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Page struct {
	Path       string
	Title      string
	IsVisible  bool
	VisitCount int
}

type DB struct {
	*sql.DB
}

func New(dataSourceName string) (*DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}

	// Create table if it doesn't exist
	query := `
    CREATE TABLE IF NOT EXISTS pages (
        path TEXT PRIMARY KEY,
        title TEXT,
        is_visible BOOLEAN NOT NULL DEFAULT TRUE,
        visit_count INTEGER NOT NULL DEFAULT 0
    );`
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func (db *DB) UpsertPage(path string) error {
	title := pathToTitle(path)
	query := `INSERT INTO pages (path, title) VALUES (?, ?) ON CONFLICT(path) DO NOTHING;`
	_, err := db.Exec(query, path, title)
	return err
}

func (db *DB) GetPageByPath(path string) (*Page, error) {
	p := &Page{}
	query := `SELECT path, title, is_visible, visit_count FROM pages WHERE path = ?;`
	err := db.QueryRow(query, path).Scan(&p.Path, &p.Title, &p.IsVisible, &p.VisitCount)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) GetAllPages() ([]Page, error) {
	rows, err := db.Query(`SELECT path, title, is_visible, visit_count FROM pages ORDER BY path;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		p := Page{}
		if err := rows.Scan(&p.Path, &p.Title, &p.IsVisible, &p.VisitCount); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, nil
}

func (db *DB) GetVisiblePages() ([]Page, error) {
	rows, err := db.Query(`SELECT path, title FROM pages WHERE is_visible = TRUE ORDER BY path;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		p := Page{}
		// We only scan path and title for nav
		if err := rows.Scan(&p.Path, &p.Title); err != nil {
			return nil, err
		}
		// Remove .md for display links
		p.Path = strings.TrimSuffix(p.Path, ".md")
		if p.Path == "home" {
			p.Path = "/"
		}
		pages = append(pages, p)
	}
	return pages, nil
}

func (db *DB) IncrementVisitCount(path string) {
	query := `UPDATE pages SET visit_count = visit_count + 1 WHERE path = ?;`
	_, err := db.Exec(query, path)
	if err != nil {
		log.Printf("Error incrementing visit count for %s: %v", path, err)
	}
}

func (db *DB) ToggleVisibility(path string) error {
	query := `UPDATE pages SET is_visible = NOT is_visible WHERE path = ?;`
	_, err := db.Exec(query, path)
	return err
}

// pathToTitle converts a file path like "some/awesome-page.md" to "Some Awesome Page".
func pathToTitle(path string) string {
	// Remove extension
	base := strings.TrimSuffix(path, filepath.Ext(path))
	// Get last part of path
	base = filepath.Base(base)
	// Replace separators with spaces
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	// Capitalize
	return strings.Title(base)
}
