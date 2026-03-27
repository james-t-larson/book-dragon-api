package store

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"

	"book-dragon/internal/models"
)

var ErrUserNotFound = errors.New("user not found")
var ErrDuplicateEmail = errors.New("a user with this email already exists")

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open sqlite db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping sqlite db: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("could not create tables: %w", err)
	}

	return &Store{db: db}, nil
}

func createTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(query)
	return err
}

func (s *Store) CreateUser(u *models.User) error {
	query := `INSERT INTO users (username, email, password) VALUES (?, ?, ?)`
	result, err := s.db.Exec(query, u.Username, u.Email, u.Password)
	if err != nil {
		// Basic check for SQLite unique constraint error
		if err.Error() == "UNIQUE constraint failed: users.email" {
			return ErrDuplicateEmail
		}
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	u.ID = id
	return nil
}

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	query := `SELECT id, username, email, password, created_at FROM users WHERE email = ?`
	row := s.db.QueryRow(query, email)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (s *Store) GetUserByID(id int64) (*models.User, error) {
	query := `SELECT id, username, email, password, created_at FROM users WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}
