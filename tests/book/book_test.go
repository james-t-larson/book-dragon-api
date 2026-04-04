package booktest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"book-dragon/internal/auth"
	"book-dragon/internal/handlers"
	"book-dragon/internal/models"
	"book-dragon/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return st
}

func setupUser(st *store.Store) *models.User {
	hashedPassword, _ := auth.HashPassword("password123")
	u := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
	}
	_ = st.CreateUser(context.Background(), u)
	return u
}

func TestPostBook(t *testing.T) {
	st := setupTestStore(t)
	u := setupUser(st)
	handler := &handlers.BookHandler{Store: st}

	tests := []struct {
		name           string
		payload        interface{}
		setContext     bool
		expectedStatus int
		verifyCount    int // expected read count after
	}{
		{
			name: "Create Book",
			payload: models.CreateBookRequest{
				Title:      "1984",
				Author:     "George Orwell",
				Genre:      "Dystopian",
				TotalPages: 328,
			},
			setContext:     true,
			expectedStatus: http.StatusCreated,
			verifyCount:    1,
		},
		{
			name: "Increment Existing Book",
			payload: models.CreateBookRequest{
				Title:      "1984",
				Author:     "George Orwell",
				Genre:      "Dystopian",
				TotalPages: 328,
			},
			setContext:     true,
			expectedStatus: http.StatusCreated,
			verifyCount:    2,
		},
		{
			name: "Missing Fields",
			payload: models.CreateBookRequest{
				Title: "1984",
			},
			setContext:     true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			payload: models.CreateBookRequest{
				Title:      "Brave New World",
				Author:     "Aldous Huxley",
				Genre:      "Dystopian",
				TotalPages: 288,
			},
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			if tc.setContext {
				ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.PostBook(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tc.expectedStatus, w.Result().StatusCode, w.Body.String())
			}

			if tc.expectedStatus == http.StatusCreated {
				var book models.Book
				json.Unmarshal(w.Body.Bytes(), &book)
				if book.ID == 0 || book.Title == "" {
					t.Fatalf("expected valid book in response, got %v", book)
				}

				// Verify read count
				userBooks, _ := st.GetUserBooks(context.Background(), u.ID)
				found := false
				for _, ub := range userBooks {
					if ub.Title == book.Title {
						found = true
						if ub.ReadCount != tc.verifyCount {
							t.Fatalf("expected read count %d, got %d", tc.verifyCount, ub.ReadCount)
						}
					}
				}
				if !found {
					t.Fatalf("user book relationship not found")
				}
			}
		})
	}
}

func TestGetBooks(t *testing.T) {
	st := setupTestStore(t)
	u := setupUser(st)
	handler := &handlers.BookHandler{Store: st}

	// Add a book
	b, _ := st.GetOrCreateBook(context.Background(), "Dune", "Frank Herbert", "Sci-Fi", 412)
	_ = st.IncrementUserBook(context.Background(), u.ID, b.ID)
	_ = st.IncrementUserBook(context.Background(), u.ID, b.ID) // Read count should be 2

	tests := []struct {
		name           string
		setContext     bool
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid Token",
			setContext:     true,
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var books []models.UserBookResponse
				if err := json.Unmarshal(body, &books); err != nil {
					t.Fatalf("failed to parse books response: %v", err)
				}
				if len(books) != 1 {
					t.Fatalf("expected 1 book, got %d", len(books))
				}
				if books[0].Title != "Dune" || books[0].ReadCount != 2 {
					t.Errorf("expected book 'Dune' with read count 2, got title %s and count %d", books[0].Title, books[0].ReadCount)
				}
				if books[0].ID == 0 {
					t.Errorf("expected book ID to be non-zero in GET /books response")
				}
			},
		},
		{
			name:           "Missing Context (Unauthorized)",
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
			checkBody:      nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/books", nil)
			if tc.setContext {
				ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.GetBooks(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Result().StatusCode)
			}

			if tc.checkBody != nil {
				tc.checkBody(t, w.Body.Bytes())
			}
		})
	}
}
