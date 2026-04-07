package store

import (
	"context"
	"database/sql"
	"errors"

	"book-dragon/internal/models"
)

func (s *Store) GetOrCreateBook(ctx context.Context, title, author, genre string, totalPages int) (*models.Book, error) {
	queryString := `SELECT id, title, author, genre, total_pages, created_at FROM books WHERE author = ? AND title = ?`
	row := s.queryRow(ctx, queryString, author, title)

	var b models.Book
	err := row.Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.TotalPages, &b.CreatedAt)
	if err == nil {
		return &b, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	insertQuery := `INSERT INTO books (title, author, genre, total_pages) VALUES (?, ?, ?, ?)`
	result, err := s.exec(ctx, insertQuery, title, author, genre, totalPages)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetBookByID(ctx, id)
}

func (s *Store) GetBookByID(ctx context.Context, id int64) (*models.Book, error) {
	queryString := `SELECT id, title, author, genre, total_pages, created_at FROM books WHERE id = ?`
	row := s.queryRow(ctx, queryString, id)

	var b models.Book
	err := row.Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.TotalPages, &b.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (s *Store) AddUserBook(ctx context.Context, userID, bookID int64) error {
	return s.AddUserBookWithReading(ctx, userID, bookID, false)
}

// AddUserBookWithReading inserts a user_book with explicit reading flag.
func (s *Store) AddUserBookWithReading(ctx context.Context, userID, bookID int64, reading bool) error {
	checkQuery := `SELECT 1 FROM user_books WHERE user_id = ? AND book_id = ?`
	var exists int
	err := s.queryRow(ctx, checkQuery, userID, bookID).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		insertQuery := `INSERT INTO user_books (user_id, book_id, read_count, reading) VALUES (?, ?, 0, ?)`
		_, err := s.exec(ctx, insertQuery, userID, bookID, reading)
		return err
	}
	return err
}

// UpdateUserBook updates reading flag and current page for a user's book.
func (s *Store) UpdateUserBook(ctx context.Context, userID, bookID int64, reading bool, currentPage int) error {
	if err := s.UpdateUserBookReading(ctx, userID, bookID, reading); err != nil {
		return err
	}
	return s.UpdateUserBookProgress(ctx, userID, bookID, currentPage)
}

func (s *Store) GetUserBooks(ctx context.Context, userID int64, currentlyReading bool) ([]models.UserBookResponse, error) {
	queryString := `
		SELECT b.id, b.title, b.author, b.genre, b.total_pages, ub.read_count, ub.current_page, ub.reading
		FROM books b
		JOIN user_books ub ON b.id = ub.book_id
		WHERE ub.user_id = ?`
	if currentlyReading {
		queryString += ` AND ub.reading = 1`
	}
	rows, err := s.query(ctx, queryString, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []models.UserBookResponse
	for rows.Next() {
		var b models.UserBookResponse
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.TotalPages, &b.ReadCount, &b.CurrentPage, &b.Reading); err != nil {
			return nil, err
		}
		books = append(books, b)
	}

	return books, nil
}

func (s *Store) GetUserBookSummaries(ctx context.Context, userID int64) ([]models.UserBookSummary, error) {
	queryString := `
		SELECT b.id, b.title, ub.read_count
		FROM books b
		JOIN user_books ub ON b.id = ub.book_id
		WHERE ub.user_id = ?
	`
	rows, err := s.query(ctx, queryString, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []models.UserBookSummary
	for rows.Next() {
		var b models.UserBookSummary
		if err := rows.Scan(&b.ID, &b.Title, &b.ReadCount); err != nil {
			return nil, err
		}
		books = append(books, b)
	}

	return books, nil
}

func (s *Store) HasUserBook(ctx context.Context, userID, bookID int64) (bool, error) {
	queryString := `SELECT 1 FROM user_books WHERE user_id = ? AND book_id = ?`
	var exists int
	err := s.queryRow(ctx, queryString, userID, bookID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Store) UpdateUserBookProgress(ctx context.Context, userID, bookID int64, currentPage int) error {
	book, err := s.GetBookByID(ctx, bookID)
	if err != nil {
		return err
	}

	readCountIncrement := 0
	finalCurrentPage := currentPage

	if currentPage >= book.TotalPages {
		readCountIncrement = 1
		finalCurrentPage = 0
		// also clear reading flag when completed
		if err := s.UpdateUserBookReading(ctx, userID, bookID, false); err != nil {
			return err
		}
	}

	queryString := `UPDATE user_books SET current_page = ?, read_count = read_count + ? WHERE user_id = ? AND book_id = ?`
	_, err = s.exec(ctx, queryString, finalCurrentPage, readCountIncrement, userID, bookID)
	return err
}

// UpdateUserBookReading sets the reading flag for a user's book.
func (s *Store) UpdateUserBookReading(ctx context.Context, userID, bookID int64, reading bool) error {
	queryString := `UPDATE user_books SET reading = ? WHERE user_id = ? AND book_id = ?`
	_, err := s.exec(ctx, queryString, reading, userID, bookID)
	return err
}
