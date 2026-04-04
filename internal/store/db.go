package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"

	"book-dragon/internal/middleware"
	"book-dragon/internal/models"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var ErrUserNotFound = errors.New("user not found")
var ErrDuplicateEmail = errors.New("a user with this email already exists")
var ErrDragonAlreadyExists = errors.New("user already has a dragon")
var ErrDragonNotFound = errors.New("dragon not found")

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

	d, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create iofs driver: %w", err)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("could not create migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		d,
		"sqlite3",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize migrate: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("could not run migrations: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if queries, ok := ctx.Value(middleware.QueriesContextKey).(*[]string); ok {
		*queries = append(*queries, query)
	}
	return s.db.ExecContext(ctx, query, args...)
}

func (s *Store) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if queries, ok := ctx.Value(middleware.QueriesContextKey).(*[]string); ok {
		*queries = append(*queries, query)
	}
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *Store) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if queries, ok := ctx.Value(middleware.QueriesContextKey).(*[]string); ok {
		*queries = append(*queries, query)
	}
	return s.db.QueryContext(ctx, query, args...)
}

func (s *Store) CreateUser(ctx context.Context, u *models.User) error {
	queryString := `INSERT INTO users (username, email, password) VALUES (?, ?, ?)`
	result, err := s.exec(ctx, queryString, u.Username, u.Email, u.Password)
	if err != nil {
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

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	queryString := `
		SELECT u.id, u.username, u.email, u.password, u.created_at, u.coins,
		       d.id, d.name, d.color
		FROM users u
		LEFT JOIN dragons d ON u.id = d.user_id
		WHERE u.email = ?
	`
	row := s.queryRow(ctx, queryString, email)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.CreatedAt, &u.Coins, &u.DragonID, &u.DragonName, &u.DragonColor)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	books, err := s.GetUserBookSummaries(ctx, u.ID)
	if err == nil && len(books) > 0 {
		u.Books = books
	} else {
		u.Books = []models.UserBookSummary{}
	}

	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	queryString := `
		SELECT u.id, u.username, u.email, u.password, u.created_at, u.coins,
		       d.id, d.name, d.color
		FROM users u
		LEFT JOIN dragons d ON u.id = d.user_id
		WHERE u.id = ?
	`
	row := s.queryRow(ctx, queryString, id)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.CreatedAt, &u.Coins, &u.DragonID, &u.DragonName, &u.DragonColor)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	books, err := s.GetUserBookSummaries(ctx, u.ID)
	if err == nil && len(books) > 0 {
		u.Books = books
	} else {
		u.Books = []models.UserBookSummary{}
	}

	return &u, nil
}

func (s *Store) CreateDragon(ctx context.Context, d *models.Dragon) error {
	queryString := `INSERT INTO dragons (name, color, user_id) VALUES (?, ?, ?)`
	result, err := s.exec(ctx, queryString, d.Name, d.Color, d.UserID)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: dragons.user_id" {
			return ErrDragonAlreadyExists
		}
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	d.ID = id
	return nil
}

func (s *Store) GetDragonByUserID(ctx context.Context, userID int64) (*models.Dragon, error) {
	queryString := `SELECT id, name, color, user_id, created_at FROM dragons WHERE user_id = ?`
	row := s.queryRow(ctx, queryString, userID)

	var d models.Dragon
	err := row.Scan(&d.ID, &d.Name, &d.Color, &d.UserID, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDragonNotFound
		}
		return nil, err
	}

	return &d, nil
}

func (s *Store) AddCoinsToUser(ctx context.Context, userID int64, coinsToAdd int64) (int64, error) {
	queryString := `UPDATE users SET coins = coins + ? WHERE id = ?`
	_, err := s.exec(ctx, queryString, coinsToAdd, userID)
	if err != nil {
		return 0, err
	}

	var totalCoins int64
	err = s.queryRow(ctx, `SELECT coins FROM users WHERE id = ?`, userID).Scan(&totalCoins)
	if err != nil {
		return 0, err
	}
	return totalCoins, nil
}
