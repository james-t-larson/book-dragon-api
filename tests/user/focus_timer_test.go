package usertest

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

func TestFocusTimerComplete(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(st *store.Store) (*models.User, *models.Book)
		payload        interface{}
		setContext     bool
		expectedStatus int
		verifyEarned   int64
		verifyTotal    int64
	}{
		{
			name: "Valid Session Earns Coins",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				hashedPassword, _ := auth.HashPassword("password123")
				u := &models.User{
					Username: "testuser",
					Email:    "test@example.com",
					Password: hashedPassword,
				}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("Dune", "Frank Herbert", "Sci-Fi", 412)
				_ = st.IncrementUserBook(u.ID, b.ID)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 15, BookID: b.ID}
			},
			setContext:     true,
			expectedStatus: http.StatusOK,
			verifyEarned:   9,
			verifyTotal:    9,
		},
		{
			name: "Less Than 5 Minutes Earns No Coins",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				hashedPassword, _ := auth.HashPassword("password123")
				u := &models.User{
					Username: "testuser2",
					Email:    "test2@example.com",
					Password: hashedPassword,
				}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("1984", "George Orwell", "Dystopian", 328)
				_ = st.IncrementUserBook(u.ID, b.ID)
				
				// Give user 5 initial coins for testing
				st.AddCoinsToUser(u.ID, 5)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 4, BookID: b.ID}
			},
			setContext:     true,
			expectedStatus: http.StatusOK,
			verifyEarned:   0,
			verifyTotal:    5,
		},
		{
			name: "Book Not In Library",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				hashedPassword, _ := auth.HashPassword("password123")
				u := &models.User{
					Username: "testuser3",
					Email:    "test3@example.com",
					Password: hashedPassword,
				}
				_ = st.CreateUser(u)
				// Create book but don't add to user library
				b, _ := st.GetOrCreateBook("Foundation", "Isaac Asimov", "Sci-Fi", 255)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 10, BookID: b.ID}
			},
			setContext:     true,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Zero Minutes",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				u := &models.User{Username: "testuser4", Email: "test4@test.com", Password: "pwd"}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("Brave New World", "Aldous Huxley", "Dystopian", 311)
				_ = st.IncrementUserBook(u.ID, b.ID)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 0, BookID: b.ID}
			},
			setContext:     true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				u := &models.User{Username: "testuser5", Email: "test5@test.com", Password: "pwd"}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("Fahrenheit 451", "Ray Bradbury", "Dystopian", 256)
				_ = st.IncrementUserBook(u.ID, b.ID)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 10, BookID: b.ID}
			},
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := setupTestStore(t)
			u, b := tc.setup(st)
			handler := &handlers.UserHandler{Store: st}

			var payload interface{}
			if fn, ok := tc.payload.(func(*models.Book) interface{}); ok {
				payload = fn(b)
			} else {
				payload = tc.payload
			}

			bodyBytes, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/focus_timer_complete", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			if tc.setContext {
				ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.FocusTimerComplete(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tc.expectedStatus, w.Result().StatusCode, w.Body.String())
			}

			if tc.expectedStatus == http.StatusOK {
				var res models.FocusTimerResponse
				if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if res.CoinsEarned != tc.verifyEarned {
					t.Errorf("expected earned %d, got %d", tc.verifyEarned, res.CoinsEarned)
				}
				if res.TotalCoins != tc.verifyTotal {
					t.Errorf("expected total %d, got %d", tc.verifyTotal, res.TotalCoins)
				}
			}
		})
	}
}
