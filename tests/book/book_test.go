package booktest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
			verifyCount:    0,
		},
		{
			name: "Add Existing Book",
			payload: models.CreateBookRequest{
				Title:      "1984",
				Author:     "George Orwell",
				Genre:      "Dystopian",
				TotalPages: 328,
			},
			setContext:     true,
			expectedStatus: http.StatusCreated,
			verifyCount:    0,
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
				userBooks, _ := st.GetUserBooks(context.Background(), u.ID, false)
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

	// Add books: one reading, one not
	b1, _ := st.GetOrCreateBook(context.Background(), "Dune", "Frank Herbert", "Sci-Fi", 412)
	_ = st.AddUserBookWithReading(context.Background(), u.ID, b1.ID, true)
	
	b2, _ := st.GetOrCreateBook(context.Background(), "1984", "George Orwell", "Dystopian", 328)
	_ = st.AddUserBookWithReading(context.Background(), u.ID, b2.ID, false)

	tests := []struct {
		name           string
		setContext     bool
		query          string
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "All Books",
			setContext:     true,
			query:          "",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var books []models.UserBookResponse
				json.Unmarshal(body, &books)
				if len(books) != 2 {
					t.Fatalf("expected 2 books, got %d", len(books))
				}
			},
		},
		{
			name:           "Currently Reading Only",
			setContext:     true,
			query:          "?currently_reading=true",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var books []models.UserBookResponse
				json.Unmarshal(body, &books)
				if len(books) != 1 {
					t.Fatalf("expected 1 book, got %d", len(books))
				}
				if books[0].Title != "Dune" {
					t.Errorf("expected Dune, got %s", books[0].Title)
				}
				if !books[0].Reading {
					t.Errorf("expected reading to be true")
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
			req := httptest.NewRequest(http.MethodGet, "/books"+tc.query, nil)
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

func TestUpdateBook(t *testing.T) {
	st := setupTestStore(t)
	u := setupUser(st)
	handler := &handlers.BookHandler{Store: st}

	b, _ := st.GetOrCreateBook(context.Background(), "Dune", "Frank Herbert", "Sci-Fi", 412)
	_ = st.AddUserBook(context.Background(), u.ID, b.ID)

	tests := []struct {
		name           string
		bookID         int64
		payload        models.UpdateBookRequest
		setContext     bool
		expectedStatus int
	}{
		{
			name:   "Set Reading True",
			bookID: b.ID,
			payload: models.UpdateBookRequest{
				Reading:     true,
				CurrentPage: 50,
			},
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Set Reading False",
			bookID: b.ID,
			payload: models.UpdateBookRequest{
				Reading:     false,
				CurrentPage: 60,
			},
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Unauthorized",
			bookID: b.ID,
			payload: models.UpdateBookRequest{
				Reading: true,
			},
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPut, "/books/", bytes.NewBuffer(bodyBytes))
			// Manually set URL path to simulate router parameter if needed, 
			// though the handler expects it at the end of the path.
			req.URL.Path = "/books/" + strconv.FormatInt(tc.bookID, 10)
			
			if tc.setContext {
				ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.UpdateBook(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tc.expectedStatus, w.Result().StatusCode, w.Body.String())
			}

			if tc.expectedStatus == http.StatusOK {
				// Verify in DB
				books, _ := st.GetUserBooks(context.Background(), u.ID, false)
				for _, ub := range books {
					if ub.ID == tc.bookID {
						if ub.Reading != tc.payload.Reading {
							t.Errorf("expected reading %v, got %v", tc.payload.Reading, ub.Reading)
						}
						if ub.CurrentPage != tc.payload.CurrentPage {
							t.Errorf("expected current_page %d, got %d", tc.payload.CurrentPage, ub.CurrentPage)
						}
					}
				}
			}
		})
	}
}
