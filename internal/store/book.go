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

func (s *Store) IncrementUserBook(ctx context.Context, userID, bookID int64) error {
	checkQuery := `SELECT id, read_count FROM user_books WHERE user_id = ? AND book_id = ?`
	row := s.queryRow(ctx, checkQuery, userID, bookID)

	var id int64
	var readCount int
	err := row.Scan(&id, &readCount)
	
	if errors.Is(err, sql.ErrNoRows) {
		insertQuery := `INSERT INTO user_books (user_id, book_id, read_count) VALUES (?, ?, 1)`
		_, err := s.exec(ctx, insertQuery, userID, bookID)
		return err
	} else if err != nil {
		return err
	}

	updateQuery := `UPDATE user_books SET read_count = read_count + 1 WHERE id = ?`
	_, err = s.exec(ctx, updateQuery, id)
	return err
}

func (s *Store) GetUserBooks(ctx context.Context, userID int64) ([]models.UserBookResponse, error) {
	queryString := `
		SELECT b.id, b.title, b.author, b.genre, b.total_pages, ub.read_count, ub.current_page
		FROM books b
		JOIN user_books ub ON b.id = ub.book_id
		WHERE ub.user_id = ?
	`
	rows, err := s.query(ctx, queryString, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []models.UserBookResponse
	for rows.Next() {
		var b models.UserBookResponse
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.TotalPages, &b.ReadCount, &b.CurrentPage); err != nil {
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

func (s *Store) AddPagesRead(ctx context.Context, userID, bookID int64, pagesRead int) error {
	queryString := `UPDATE user_books SET current_page = current_page + ? WHERE user_id = ? AND book_id = ?`
	_, err := s.exec(ctx, queryString, pagesRead, userID, bookID)
	return err
}
