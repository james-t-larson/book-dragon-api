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

func intPtr(i int) *int { return &i }

func TestFocusTimerComplete(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(st *store.Store) (*models.User, *models.Book)
		payload          interface{}
		setContext       bool
		expectedStatus   int
		verifyEarned     int64
		verifyTotal      int64
		verifyPagesMoved int
	}{
		{
			name: "Valid Session Earns Coins With Pages",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				hashedPassword, _ := auth.HashPassword("password123")
				u := &models.User{Username: "testuser", Email: "test@example.com", Password: hashedPassword}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("Dune", "Frank Herbert", "Sci-Fi", 412)
				_ = st.IncrementUserBook(u.ID, b.ID)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 15, BookID: b.ID, PagesRead: intPtr(20)}
			},
			setContext:       true,
			expectedStatus:   http.StatusOK,
			verifyEarned:     9,
			verifyTotal:      9,
			verifyPagesMoved: 20,
		},
		{
			name: "Valid Session Zero Pages Read",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				hashedPassword, _ := auth.HashPassword("pass")
				u := &models.User{Username: "testuser6", Email: "test6@example.com", Password: hashedPassword}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("Neuromancer", "William Gibson", "Sci-Fi", 271)
				_ = st.IncrementUserBook(u.ID, b.ID)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 10, BookID: b.ID, PagesRead: intPtr(0)}
			},
			setContext:     true,
			expectedStatus: http.StatusOK,
			verifyEarned:   6,
			verifyTotal:    6,
		},
		{
			name: "Less Than 5 Minutes Earns No Coins",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				hashedPassword, _ := auth.HashPassword("password123")
				u := &models.User{Username: "testuser2", Email: "test2@example.com", Password: hashedPassword}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("1984", "George Orwell", "Dystopian", 328)
				_ = st.IncrementUserBook(u.ID, b.ID)
				st.AddCoinsToUser(u.ID, 5)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 4, BookID: b.ID, PagesRead: intPtr(3)}
			},
			setContext:       true,
			expectedStatus:   http.StatusOK,
			verifyEarned:     0,
			verifyTotal:      5,
			verifyPagesMoved: 3,
		},
		{
			name: "Pages Read Omitted (Required)",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				u := &models.User{Username: "testuser8", Email: "test8@test.com", Password: "pwd"}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("Dune Messiah", "Frank Herbert", "Sci-Fi", 331)
				_ = st.IncrementUserBook(u.ID, b.ID)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				// Deliberately omit pages_read by using a raw map
				return map[string]interface{}{"minutes": 10, "book_id": b.ID}
			},
			setContext:     true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Book Not In Library",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				hashedPassword, _ := auth.HashPassword("password123")
				u := &models.User{Username: "testuser3", Email: "test3@example.com", Password: hashedPassword}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("Foundation", "Isaac Asimov", "Sci-Fi", 255)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 10, BookID: b.ID, PagesRead: intPtr(5)}
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
				return models.FocusTimerRequest{Minutes: 0, BookID: b.ID, PagesRead: intPtr(0)}
			},
			setContext:     true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Negative Pages Read",
			setup: func(st *store.Store) (*models.User, *models.Book) {
				u := &models.User{Username: "testuser7", Email: "test7@test.com", Password: "pwd"}
				_ = st.CreateUser(u)
				b, _ := st.GetOrCreateBook("The Road", "Cormac McCarthy", "Post-Apocalyptic", 287)
				_ = st.IncrementUserBook(u.ID, b.ID)
				return u, b
			},
			payload: func(b *models.Book) interface{} {
				return models.FocusTimerRequest{Minutes: 10, BookID: b.ID, PagesRead: intPtr(-5)}
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
				return models.FocusTimerRequest{Minutes: 10, BookID: b.ID, PagesRead: intPtr(10)}
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

				if tc.verifyPagesMoved > 0 {
					books, err := st.GetUserBooks(u.ID)
					if err != nil {
						t.Fatalf("failed to get user books: %v", err)
					}
					var found bool
					for _, bk := range books {
						if bk.ID == b.ID {
							found = true
							if bk.CurrentPage != tc.verifyPagesMoved {
								t.Errorf("expected current_page %d, got %d", tc.verifyPagesMoved, bk.CurrentPage)
							}
						}
					}
					if !found {
						t.Errorf("book not found in user's library after session")
					}
				}
			}
		})
	}
}
