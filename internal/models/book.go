package models

import "time"

type Book struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	Genre      string    `json:"genre"`
	TotalPages int       `json:"total_pages"`
	CreatedAt  time.Time `json:"created_at"`
}

type UserBook struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	BookID      int64     `json:"book_id"`
	ReadCount   int       `json:"read_count"`
	CurrentPage int       `json:"current_page"`
	Reading     bool      `json:"reading"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateBookRequest struct {
	Title      string `json:"title"`
	Author     string `json:"author"`
	Genre      string `json:"genre"`
	TotalPages int    `json:"total_pages"`
	Reading    bool   `json:"reading"`
}

type UserBookSummary struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	ReadCount int    `json:"read_count"`
}

// UpdateBookRequest is used for PUT /books to modify reading status and current page.
type UpdateBookRequest struct {
	Reading     bool `json:"reading"`
	CurrentPage int  `json:"current_page"`
}

type UserBookResponse struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Genre       string `json:"genre"`
	TotalPages  int    `json:"total_pages"`
	ReadCount   int    `json:"read_count"`
	CurrentPage int    `json:"current_page"`
	Reading     bool   `json:"reading"`
}

