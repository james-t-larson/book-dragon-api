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
	CreatedAt   time.Time `json:"created_at"`
}

type CreateBookRequest struct {
	Title      string `json:"title"`
	Author     string `json:"author"`
	Genre      string `json:"genre"`
	TotalPages int    `json:"total_pages"`
}

type UserBookResponse struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Genre       string `json:"genre"`
	TotalPages  int    `json:"total_pages"`
	ReadCount   int    `json:"read_count"`
	CurrentPage int    `json:"current_page"`
}

type UserBookSummary struct {
	Title     string `json:"title"`
	ReadCount int    `json:"read_count"`
}
