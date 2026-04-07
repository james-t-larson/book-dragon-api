package handlers

import (
	"strconv"
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
	if err := h.Store.AddUserBookWithReading(r.Context(), userID, book.ID, req.Reading); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user book relationship")
		return
	}

	writeJSON(w, http.StatusCreated, book)
}

// @Summary Update a book
// @Description Update a user's book status
// @Tags books
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Book ID"
// @Param body body models.UpdateBookRequest true "Update info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /books/{id} [put]
func (h *BookHandler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract ID from URL (simplified assumption of router implementation)
	idStr := r.URL.Path[len("/books/"):]
	bookID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid book ID")
		return
	}

	var req models.UpdateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.Store.UpdateUserBook(r.Context(), userID, bookID, req.Reading, req.CurrentPage); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update book")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "book updated"})
}

// @Summary Get user's books
// @Description Get all books for the currently authenticated user
// @Tags books
// @Produce json
// @Security BearerAuth
// @Param currently_reading query bool false "Filter by currently reading"
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

	currentlyReadingStr := r.URL.Query().Get("currently_reading")
	currentlyReading, _ := strconv.ParseBool(currentlyReadingStr)

	books, err := h.Store.GetUserBooks(r.Context(), userID, currentlyReading)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get books")
		return
	}

	if books == nil {
		books = []models.UserBookResponse{}
	}

	writeJSON(w, http.StatusOK, books)
}
