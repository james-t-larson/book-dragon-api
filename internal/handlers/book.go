package handlers

import (
	"encoding/json"
	"net/http"

	"book-dragon/internal/auth"
	"book-dragon/internal/models"
	"book-dragon/internal/store"
)

type BookHandler struct {
	Store *store.Store
}

// @Summary Add a book
// @Description Find and add a book to the user's books, or create it if not found. Increments read count if already added.
// @Tags books
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.CreateBookRequest true "Book info"
// @Success 201 {object} models.Book
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /books [post]
func (h *BookHandler) PostBook(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.CreateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Title == "" || req.Author == "" || req.Genre == "" || req.TotalPages <= 0 {
		writeError(w, http.StatusBadRequest, "title, author, genre, and positive total_pages are required")
		return
	}

	// 1. Get or Create Book
	book, err := h.Store.GetOrCreateBook(r.Context(), req.Title, req.Author, req.Genre, req.TotalPages)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process book")
		return
	}

	// 2. Ensure UserBook relationship exists
	if err := h.Store.AddUserBook(r.Context(), userID, book.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user book relationship")
		return
	}

	writeJSON(w, http.StatusCreated, book)
}

// @Summary Get user's books
// @Description Get all books for the currently authenticated user
// @Tags books
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.UserBookResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /books [get]
func (h *BookHandler) GetBooks(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	books, err := h.Store.GetUserBooks(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get books")
		return
	}

	if books == nil {
		books = []models.UserBookResponse{}
	}

	writeJSON(w, http.StatusOK, books)
}
