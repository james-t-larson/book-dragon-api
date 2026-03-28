package store

import (
	"database/sql"
	"errors"

	"book-dragon/internal/models"
)

func (s *Store) GetOrCreateBook(title, author, genre string, totalPages int) (*models.Book, error) {
	// Try to find the book
	query := `SELECT id, title, author, genre, total_pages, created_at FROM books WHERE author = ? AND title = ?`
	row := s.db.QueryRow(query, author, title)

	var b models.Book
	err := row.Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.TotalPages, &b.CreatedAt)
	if err == nil {
		return &b, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Insert if not found
	insertQuery := `INSERT INTO books (title, author, genre, total_pages) VALUES (?, ?, ?, ?)`
	result, err := s.db.Exec(insertQuery, title, author, genre, totalPages)
	if err != nil {
		// Just in case of concurrent insert
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetBookByID(id)
}

func (s *Store) GetBookByID(id int64) (*models.Book, error) {
	query := `SELECT id, title, author, genre, total_pages, created_at FROM books WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var b models.Book
	err := row.Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.TotalPages, &b.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (s *Store) IncrementUserBook(userID, bookID int64) error {
	// Try to find the user book
	checkQuery := `SELECT id, read_count FROM user_books WHERE user_id = ? AND book_id = ?`
	row := s.db.QueryRow(checkQuery, userID, bookID)

	var id int64
	var readCount int
	err := row.Scan(&id, &readCount)
	
	if errors.Is(err, sql.ErrNoRows) {
		// Not found, insert with read_count = 1
		insertQuery := `INSERT INTO user_books (user_id, book_id, read_count) VALUES (?, ?, 1)`
		_, err := s.db.Exec(insertQuery, userID, bookID)
		return err
	} else if err != nil {
		return err
	}

	// Found, increment read_count
	updateQuery := `UPDATE user_books SET read_count = read_count + 1 WHERE id = ?`
	_, err = s.db.Exec(updateQuery, id)
	return err
}

func (s *Store) GetUserBooks(userID int64) ([]models.UserBookResponse, error) {
	query := `
		SELECT b.title, b.author, b.genre, b.total_pages, ub.read_count, ub.current_page
		FROM books b
		JOIN user_books ub ON b.id = ub.book_id
		WHERE ub.user_id = ?
	`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []models.UserBookResponse
	for rows.Next() {
		var b models.UserBookResponse
		if err := rows.Scan(&b.Title, &b.Author, &b.Genre, &b.TotalPages, &b.ReadCount, &b.CurrentPage); err != nil {
			return nil, err
		}
		books = append(books, b)
	}

	return books, nil
}

func (s *Store) GetUserBookSummaries(userID int64) ([]models.UserBookSummary, error) {
	query := `
		SELECT b.title, ub.read_count
		FROM books b
		JOIN user_books ub ON b.id = ub.book_id
		WHERE ub.user_id = ?
	`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []models.UserBookSummary
	for rows.Next() {
		var b models.UserBookSummary
		if err := rows.Scan(&b.Title, &b.ReadCount); err != nil {
			return nil, err
		}
		books = append(books, b)
	}

	return books, nil
}
